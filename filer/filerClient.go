package filer

import (
	"io/ioutil"

	"github.com/chrislusf/seaweedfs/weed/security"
	"bytes"
	"mime/multipart"
	"net/textproto"
	"fmt"
	"mime"
	"strings"
	"path/filepath"
	"io"
	"github.com/jimuyida/glog"
	"net/http"
	"encoding/json"
	"errors"
	"os"
	"path"
	"strconv"
)

type UploadResult struct {
	Name  string `json:"name,omitempty"`
	Size  uint32 `json:"size,omitempty"`
	Error string `json:"error,omitempty"`
}

var fileNameEscaper = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")

func isGzipped(filename string) bool {
	return strings.ToLower(path.Ext(filename)) == ".gz"
}

func detectMimeType(f *os.File) string {
	head := make([]byte, 512)
	f.Seek(0, 0)
	n, err := f.Read(head)
	if err == io.EOF {
		return ""
	}
	if err != nil {
		fmt.Printf("read head of %v: %v\n", f.Name(), err)
		return "application/octet-stream"
	}
	f.Seek(0, 0)
	mimeType := http.DetectContentType(head[:n])
	return mimeType
}

func CheckFile(uploadUrl string) int  {
	req, err := http.NewRequest("GET", uploadUrl,nil)
	if err != nil {
		return -1
	}

	resp, post_err := http.DefaultClient.Do(req)
	if post_err != nil {
		glog.V(0).Infoln("failing to upload to", uploadUrl, post_err.Error())
		return  -1
	}
	defer resp.Body.Close()
	clen := resp.Header.Get("Content-Length")
	len,err := strconv.Atoi(clen)
	if err != nil {
		return -1
	}
	return len
}


func DownloadFile(destPath,url string) error  {
	req, err := http.NewRequest("GET", url,nil)
	if err != nil {
		return err
	}

	resp, post_err := http.DefaultClient.Do(req)
	if post_err != nil {
		glog.V(0).Infoln("failing to upload to", url, post_err.Error())
		return  err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("code:%d",resp.StatusCode)
	}

	f,err := os.Create(destPath)
	if err != nil{
		return err
	}

	defer f.Close()

	clen := resp.Header.Get("Content-Length")
	len,err := strconv.Atoi(clen)

	sz,err := io.Copy(f,resp.Body)

	if len != int(sz) {
		os.Remove(destPath)
		return fmt.Errorf("%d != %d",len,sz)
	}

	return err
}

func DeleteFile(uploadUrl string) error  {
	req, err := http.NewRequest("DELETE", uploadUrl,nil)
	if err != nil {
		return err
	}

	resp, post_err := http.DefaultClient.Do(req)
	if post_err != nil {
		glog.V(0).Infoln("failing to upload to", uploadUrl, post_err.Error())
		return  post_err
	}
	defer resp.Body.Close()
	_, ra_err := ioutil.ReadAll(resp.Body)
	if ra_err != nil {
		return   ra_err
	}

	return nil
}

func UploadFile(uploadUrl string, filePath string, checkSameSize bool) (*UploadResult, error) {
	info,err := os.Stat(filePath)
	if err != nil  {
		fmt.Println(err)
		return nil,err
	}
	sz := CheckFile(uploadUrl)

//	fmt.Println(info.Size())
	if sz != -1 {
		if checkSameSize && int64(sz) == info.Size() {
			return &UploadResult{filepath.Base(filePath),uint32(info.Size()),""},nil
		}
	}


	err = DeleteFile(uploadUrl)
	if err != nil  {
		fmt.Println(err)
		return nil,err
	}

	f,err := os.Open(filePath)
	if err != nil {

		return nil,err
	}
	defer f.Close()
	
	return Upload(uploadUrl,filepath.Base(filePath),f,isGzipped(filepath.Base(filePath)),detectMimeType(f),nil,"")
}


func Upload(uploadUrl string, filename string, reader io.Reader, isGzipped bool, mtype string, pairMap map[string]string, jwt security.EncodedJwt) (*UploadResult, error) {
	return upload_content(uploadUrl, func(w io.Writer) (err error) {
		_, err = io.Copy(w, reader)
		return
	}, filename, isGzipped, mtype, pairMap, jwt)
}

func upload_content(uploadUrl string, fillBufferFunction func(w io.Writer) error, filename string, isGzipped bool, mtype string, pairMap map[string]string, jwt security.EncodedJwt) (*UploadResult, error) {
	body_buf := bytes.NewBufferString("")
	body_writer := multipart.NewWriter(body_buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileNameEscaper.Replace(filename)))
	if mtype == "" {
		mtype = mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	}
	if mtype != "" {
		h.Set("Content-Type", mtype)
	}
	if isGzipped {
		h.Set("Content-Encoding", "gzip")
	}
	if jwt != "" {
		h.Set("Authorization", "BEARER "+string(jwt))
	}

	file_writer, cp_err := body_writer.CreatePart(h)
	if cp_err != nil {
		glog.V(0).Infoln("error creating form file", cp_err.Error())
		return nil, cp_err
	}
	if err := fillBufferFunction(file_writer); err != nil {
		glog.V(0).Infoln("error copying data", err)
		return nil, err
	}
	content_type := body_writer.FormDataContentType()
	if err := body_writer.Close(); err != nil {
		glog.V(0).Infoln("error closing body", err)
		return nil, err
	}

	req, postErr := http.NewRequest("POST", uploadUrl, body_buf)
	if postErr != nil {
		glog.V(0).Infoln("failing to upload to", uploadUrl, postErr.Error())
		return nil, postErr
	}
	req.Header.Set("Content-Type", content_type)
	for k, v := range pairMap {
		req.Header.Set(k, v)
	}
	resp, post_err := http.DefaultClient.Do(req)
	if post_err != nil {
		glog.V(0).Infoln("failing to upload to", uploadUrl, post_err.Error())
		return nil, post_err
	}
	defer resp.Body.Close()
	resp_body, ra_err := ioutil.ReadAll(resp.Body)
	if ra_err != nil {
		return nil, ra_err
	}
	var ret UploadResult
	unmarshal_err := json.Unmarshal(resp_body, &ret)
	if unmarshal_err != nil {
		fmt.Println(string(resp_body))
		glog.V(0).Infoln("failing to read upload response", uploadUrl, string(resp_body))
		return nil, unmarshal_err
	}
	if ret.Error != "" {
		return nil, errors.New(ret.Error)
	}
	return &ret, nil
}
