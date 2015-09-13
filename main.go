package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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
	Gorelease struct {
		Host  string `gcfg:"host"`
		Token string `gcfg:"token"`
	}
}

var cfg Config

func genUptoken(bucket, key string) string {
	gr := cfg.Gorelease
	if gr.Token != "" {
		u := url.URL{
			Scheme: "http",
			Host:   gr.Host,
			Path:   "/uptoken",
		}
		query := u.Query()
		query.Set("private_token", gr.Token)
		query.Set("bucket", bucket)
		query.Set("key", key)
		u.RawQuery = query.Encode()
		log.Println(u.String())
		resp, err := http.Get(u.String())
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		uptoken, _ := ioutil.ReadAll(resp.Body)
		return string(uptoken)
	}
	putPolicy := rs.PutPolicy{
		Scope: bucket + ":" + key,
	}
	return putPolicy.Token(nil)
}

func uploadFile(bucket, key, filename string) error {
	uptoken := genUptoken(bucket, key) // in order to rewrite exists file

	var ret io.PutRet
	var extra = &io.PutExtra{}
	return io.PutFile(nil, &ret, uptoken, key, filename, extra)
}

func syncDir(bucket, keyPrefix, dir string) int {
	keyPrefix = strings.TrimPrefix(keyPrefix, "/")
	errCount := 0
	wg := &sync.WaitGroup{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		key := filepath.Join(keyPrefix, rel)
		wg.Add(1)
		go func() {
			log.Printf("Upload %v ...", strconv.Quote(key))
			if err := uploadFile(bucket, key, path); err != nil {
				errCount += 1
				log.Printf("Failed %v, %v", strconv.Quote(path), err)
			} else {
				log.Printf("Done %v", strconv.Quote(key))
			}
			wg.Done()
		}()
		return nil
	})
	wg.Wait()
	return errCount
}

func main() {
	cfgFile := flag.String("c", "conf.ini", "config file")
	flag.Parse()

	if err := gcfg.ReadFileInto(&cfg, *cfgFile); err != nil {
		log.Fatal(err)
	}
	conf.ACCESS_KEY = cfg.Qiniu.AccessKey
	conf.SECRET_KEY = cfg.Qiniu.SecretKey
	conf.UP_HOST = cfg.Qiniu.UpHost
	log.Printf("Use upload host: %v", conf.UP_HOST)

	errcnt := syncDir(cfg.Qiniu.Bucket, cfg.Qiniu.KeyPrefix, cfg.Local.SyncDir)
	if errcnt != 0 {
		log.Println("Failed count =", errcnt)
		os.Exit(1)
	}
}
