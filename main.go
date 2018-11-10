package main

import (
	"fmt"
	"os"
	"net/http"
	"strconv"
	"io"
	"github.com/disintegration/imaging"
	"image/color"
	"image"
	"./filer"
	"path/filepath"
	"strings"
)

func main()  {
	//http://file2.jimuyida.cn/tiles/55bfe3ede6784ee8a7c1e5d66524ec33/4k_face2
	locationId := []string{"55bfe3ede6784ee8a7c1e5d66524ec33","89ea2e771a1842a38a7b7902b2e2f999"}
	for _, value := range locationId {
		//preImage(value,2,4096)
		uploadImage(value)
	}

	fmt.Println("over")
}

func uploadImage(locationId string)  {
	TempDir := "./temp/big/"+locationId

	filepath.Walk(TempDir+"/", func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if !info.IsDir() {
				if ok, _ := filepath.Match("*.jpg", filepath.Base(path)); !ok {
					return nil
				}

				fmt.Print("+")
				fmt.Println(path,strings.Replace(filepath.Base(path),filepath.Ext(path),"",1))
				faces := strings.Replace(filepath.Base(path),filepath.Ext(path),"",1)

				SaveTileFile(path,faces,"./temp/"+locationId+"/",true)
			}
		} else {
			fmt.Println(err)
		}
		return err
	})
	filerHost := "http://61.183.65.149:19334"
	UploadTile("./temp/",locationId,filerHost)
}

func SaveTileFile(tileFile ,tileName, outPutDir string,isBigData bool) error {

	err := os.MkdirAll(outPutDir, os.FileMode(os.ModePerm))

	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	err = os.Mkdir(outPutDir+"/tiles", os.FileMode(os.ModePerm))

	os.MkdirAll(outPutDir+"/tiles_mobile",os.ModePerm)

	tileImage,err := imaging.Open(tileFile)
	if err != nil {
		return err
	}

	tempF,err := os.Open(tileFile)
	if err != nil {
		return err
	}
	defer tempF.Close()
	info,_,err := image.DecodeConfig(tempF)
	if err != nil {
		return err
	}


	var mobile_quality = imaging.JPEGQuality(80)

	src512 := imaging.Resize(tileImage, 512, 0, imaging.Lanczos)
	imaging.Save(src512, filepath.Join(outPutDir, "tiles/512_"+tileName+".jpg"),imaging.JPEGQuality(100))
	imaging.Save(src512, filepath.Join(outPutDir, "tiles_mobile/512_"+tileName+".jpg"),mobile_quality)
	err = imaging.Save(src512, filepath.Join(outPutDir, "tiles/512_face"+tileName+"_0_0.jpg"),imaging.JPEGQuality(100))
	if isBigData {
		err = imaging.Save(src512, filepath.Join(outPutDir, "tiles_mobile/512_face"+tileName+"_0_0.jpg"),mobile_quality)
	}




	src51k := imaging.Resize(tileImage, 1024, 0, imaging.Lanczos)
	imaging.Save(src51k, filepath.Join(outPutDir, "tiles/1k_"+tileName+".jpg"),imaging.JPEGQuality(100))
	imaging.Save(src51k, filepath.Join(outPutDir, "tiles_mobile/1k_"+tileName+".jpg"),mobile_quality)
	for  i := 0; i < 2;i++ {
		for  j := 0; j < 2;j++ {
			src512 = imaging.Crop(src51k,image.Rectangle{Min:image.Point{i*512,j*512},Max:image.Point{(i+1)*512,(j+1)*512}})
			err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles/1k_face"+tileName+"_%d_%d.jpg",i,j)),imaging.JPEGQuality(100))
			if isBigData {
				err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles_mobile/1k_face"+tileName+"_%d_%d.jpg",i,j)),mobile_quality)
			}
		}
	}

	var src2k *image.NRGBA
	if info.Width == 2048 {
		src2k = imaging.Clone(tileImage)
	} else {
		src2k = imaging.Resize(tileImage, 2048, 0, imaging.Lanczos)
	}

	imaging.Save(src2k, filepath.Join(outPutDir, "tiles/2k_"+tileName+".jpg"),imaging.JPEGQuality(100))
	imaging.Save(src2k, filepath.Join(outPutDir, "tiles_mobile/2k_"+tileName+".jpg"),mobile_quality)
	for  i := 0; i < 4;i++ {
		for  j := 0; j < 4;j++ {
			src512 = imaging.Crop(src2k,image.Rectangle{Min:image.Point{i*512,j*512},Max:image.Point{(i+1)*512,(j+1)*512}})
			err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles/2k_face"+tileName+"_%d_%d.jpg",i,j)),imaging.JPEGQuality(100))
		}
	}

	if info.Width >= 4096 {
		var src4k *image.NRGBA
		if info.Width == 4096 {
			src4k = imaging.Clone(tileImage)
		} else {
			src4k = imaging.Resize(tileImage, 4096, 0, imaging.Lanczos)
		}

		imaging.Save(src4k, filepath.Join(outPutDir, "tiles/4k_"+tileName+".jpg"),imaging.JPEGQuality(100))
		for  i := 0; i < 8;i++ {
			for  j := 0; j < 8;j++ {
				src512 = imaging.Crop(src4k,image.Rectangle{Min:image.Point{i*512,j*512},Max:image.Point{(i+1)*512,(j+1)*512}})
				err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles/4k_face"+tileName+"_%d_%d.jpg",i,j)),imaging.JPEGQuality(100))
			}
		}
	}



	return nil
}

func UploadTile(cacheDir, imageID, filerUrl string) (error) {
	var retError error = nil


	filepath.Walk(cacheDir+imageID+"/", func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if !info.IsDir() {
				if ok, _ := filepath.Match("*.jpg", filepath.Base(path)); !ok {
					return nil
				}

				fmt.Print("+")

				_, e := filer.UploadFile(filerUrl+"/tiles/"+imageID+"/"+filepath.Base(path), path, true)
				if e != nil {
					retError = e
					return e
				}

			}
		} else {
			retError = err
			fmt.Println(err)
		}
		return err
	})
	//bytes, _ := json.Marshal(retData)

	//fmt.Println(string(bytes))
	return retError
}


func preImage(locationId string,face int ,size int)  {

	filerHost := "http://61.183.65.149:19334"
	TempDir := "./temp"
	os.MkdirAll(TempDir+"/tiles/"+locationId,os.ModePerm)
	//http://file2.jimuyida.cn
	filenameTemp := "%s/4k_face%d_%d_%d.jpg"
	if size == 4096 {
		filenameTemp = "%s/4k_face%d_%d_%d.jpg"
	} else if size == 2048 {
		filenameTemp = "%s/2k_face%d_%d_%d.jpg"
	} else {
		fmt.Println("无效的图片大小")
		return
	}

	urlTemp := filerHost + "/tiles/"+filenameTemp//4k_face2_3_7.jpg

	min := 0
	max := 6

	if face > -1 {
		min = face
		max = face + 1
	}
	for i := min; i < max;i++  {
		for k := 0; k < 8;k++  {
			for j := 0; j < 8;j++  {
				filename := fmt.Sprintf(filenameTemp,locationId,i,k,j)

				url := fmt.Sprintf(urlTemp,locationId,i,k,j)

				trynum := 0
				for ; trynum < 10; trynum++  {
					fmt.Print(".")
					if !downloadTile(TempDir+"/tiles/"+filename, url) {
						continue
					} else {
						break
					}
				}
				if trynum >= 10 {
					fmt.Println("no..... ./temp/tiles/"+locationId+"/"+filename)
					//panic(fmt.Errorf("no..... ./temp/tiles/"+sid+"/"+filename))
					return
				}
			}


		}
		combineImage(locationId,i,size)
	}

	os.RemoveAll(TempDir+"/tiles/")
}

func combineImage(locationId string,face,size int)  {
	mainImg := imaging.New(size,size,color.White)
	TempDir := "./temp"

	for k := 0; k < 8;k++ {
		for j := 0; j < 8; j++ {
			filename := fmt.Sprintf("%s/4k_face%d_%d_%d.jpg",locationId,face,k,j)

			src,err := imaging.Open(TempDir+"/tiles/"+filename)
			if err != nil {
				fmt.Println(err)
				return
			}
			mainImg = imaging.Paste(mainImg,src,image.Point{k*512,j*512})
		}
	}
	os.MkdirAll(TempDir+"/big/"+locationId,os.ModePerm)
	filename := fmt.Sprintf("%s/%d.jpg",locationId,face)
	imaging.Save(mainImg,TempDir+"/big/"+filename)

}

func downloadTile(dataFile, url string) (result bool) {
	result = false

	defer func() {
		if !result {
			os.Remove(dataFile)
		}

	}()
	_, err := os.Stat(dataFile)
	if err == nil {
		//ctx.SendFile(imgFile,ctx.Params().Get("imgId"))
		result = true
		return
	}

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		fmt.Println(err)

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		result = true
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(err, resp.StatusCode, url)
		return
	}

	clen := resp.Header.Get("Content-Length")
	len,_ := strconv.Atoi(clen)

	f, err := os.Create(dataFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	sz, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	if sz != int64(len) {
		fmt.Println(sz,len)
		return
	}

	result = true

	return
}
