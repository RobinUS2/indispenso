package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/RobinUS2/golang-jresp"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"net/http"
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
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	zw := zip.NewWriter(buf)

	// Add some files to the archive.
	var files = []struct {
		Name string
	}{
		{"users.json"},
		{"templates.conf"},
		{"httpchecks.json"},
	}
	for _, file := range files {
		// Create file in zip archive
		f, err := zw.Create(file.Name)
		if err != nil {
			jr.Error(fmt.Sprintf("Failed creating zip: %s", err))
			fmt.Fprint(w, jr.ToString(debug))
			return
		}

		// Read contents from file\
		fileB, fileE := ioutil.ReadFile(fmt.Sprintf("/etc/indispenso/%s", file.Name))
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
	w.Header().Set("Content-Disposition", "attachment; filename=indispenso_backup.zip")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf.Bytes())))

	// Dump as download
	io.Copy(w, buf)
}
