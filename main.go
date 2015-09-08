package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/qiniu/api.v6/conf"
	"github.com/qiniu/api.v6/io"
	"github.com/qiniu/api.v6/rs"
	"gopkg.in/gcfg.v1"
)

type Config struct {
	Qiniu struct {
		UpHost    string `gcfg:"uphost"`
		AccessKey string `gcfg:"accesskey"`
		SecretKey string `gcfg:"secretkey"`
		Bucket    string
		KeyPrefix string `gcfg:"keyprefix"`
	}
	Local struct {
		SyncDir string `gcfg:"syncdir"`
	}
}

func genUptoken(bucketName string) string {
	putPolicy := rs.PutPolicy{
		Scope: bucketName,
	}
	//putPolicy.SaveKey = key
	return putPolicy.Token(nil)
}

func uploadFile(bucket, key, filename string) error {
	uptoken := genUptoken(bucket + ":" + key) // in order to rewrite exists file

	var ret io.PutRet
	var extra = &io.PutExtra{}
	return io.PutFile(nil, &ret, uptoken, key, filename, extra)
}

func syncDir(bucket, keyPrefix, dir string) int {
	keyPrefix = strings.TrimPrefix(keyPrefix, "/")
	errCount := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		key := filepath.Join(keyPrefix, rel)
		log.Printf("upload to %v ...", strconv.Quote(key))
		if err := uploadFile(bucket, key, path); err != nil {
			errCount += 1
			log.Println(path, info)
		}
		return nil
	})
	return errCount
}

func main() {
	cfgFile := flag.String("c", "conf.ini", "config file")
	flag.Parse()

	var cfg Config
	if err := gcfg.ReadFileInto(&cfg, *cfgFile); err != nil {
		log.Fatal(err)
	}
	conf.ACCESS_KEY = cfg.Qiniu.AccessKey
	conf.SECRET_KEY = cfg.Qiniu.SecretKey
	conf.UP_HOST = cfg.Qiniu.UpHost

	syncDir(cfg.Qiniu.Bucket, cfg.Qiniu.KeyPrefix, cfg.Local.SyncDir)
	/*
		finfos, err := ioutil.ReadDir(cfg.Local.SyncDir)
		if err != nil {
			log.Fatal(err)
		}
		for _, finfo := range finfos {
			log.Printf("Upload %v ...", finfo.Name())
			log.Println(finfo)
		}
		if err := uploadFile(cfg.Qiniu.Bucket, "gorelease/", "main.go"); err != nil {
			log.Println(err)
		}
	*/
}
