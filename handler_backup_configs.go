package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/RobinUS2/golang-jresp"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

// Backup data from the server in a ZIP file
func GetBackupConfigs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Logged in
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Must be admin
	usr := getUser(r)
	if !usr.HasRole("admin") {
		jr.Error("Not allowed")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	zw := zip.NewWriter(buf)

	// Add some files to the archive.
	//TOOD create struct and add files form respective modules
	var files = []struct {
		Name string
	}{
		{fmt.Sprintf("%s/%s", conf.GetHome(), "users.json")},
		{fmt.Sprintf("%s/%s", conf.GetHome(), "templates.conf")},
		{fmt.Sprintf("%s/%s", conf.GetHome(), "httpchecks.json")},
		{conf.GetSslCertFile()},
		{conf.GetSslPrivateKeyFile()},
		{conf.ConfFile()},
	}
	for _, file := range files {
		fileName := file.Name
		fmt.Println(file.Name)
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			continue
		}

		// Create file in zip archive
		f, err := zw.Create(path.Base(file.Name))
		if err != nil {
			jr.Error(fmt.Sprintf("Failed creating zip: %s", err))
			fmt.Fprint(w, jr.ToString(debug))
			return
		}

		// Read contents from file\
		fileB, fileE := ioutil.ReadFile(fileName)
		if fileE != nil {
			jr.Error(fmt.Sprintf("Failed creating zip: %s", fileE))
			fmt.Fprint(w, jr.ToString(debug))
			return
		}

		// Write into file
		_, err = f.Write(fileB)
		if err != nil {
			jr.Error(fmt.Sprintf("Failed creating zip: %s", err))
			fmt.Fprint(w, jr.ToString(debug))
			return
		}
	}

	// Make sure to check the error on Close.
	zw.Flush()
	err := zw.Close()
	if err != nil {
		jr.Error(fmt.Sprintf("Failed creating zip: %s", err))
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename=\"indispenso.zip\"")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf.Bytes())))

	// Dump as download
	w.Write(buf.Bytes())
}
