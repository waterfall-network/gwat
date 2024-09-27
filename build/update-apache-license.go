// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build none
// +build none

/*
This command generates Apache license headers on top of all source files.
You can run it once per month, before cutting a release or just
whenever you feel like it.

	go run update-license.go

All authors (people who have contributed code) are listed in the
AUTHORS file. The author names are mapped and deduplicated using the
.mailmap file. You can use .mailmap to set the canonical name and
address for each author. See git-shortlog(1) for an explanation of the
.mailmap format.

Please review the resulting diff to check whether the correct
copyright assignments are performed.
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
)

var (
	// only files with these extensions will be considered
	extensions = []string{".go"}

	// paths with any of these prefixes will be skipped
	skipPrefixes = []string{
		"vendor/", "tests/testdata/", "build/",
		"cmd/internal/browser", "common/bitutil/bitutil",
		"crypto/bn256/", "crypto/ecies/", "graphql/graphiql.go",
		"crypto/bls12381/", "metrics/", "log/",
		"crypto/bls_sig/", "crypto/secp256k1",
	}

	licenseCommentRE = regexp.MustCompile(`^//\s*(Copyright|This file is part of).*?\n(?://.*?\n)*\n*`)
)

var licenseT = template.Must(template.New("").Parse(`
// Copyright 2024   Blue Wave Inc.
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

`[1:]))

type info struct {
	file string
	Year int64
}

func main() {
	var (
		files = getFiles()
		filec = make(chan string)
		infoc = make(chan *info, 20)
		wg    sync.WaitGroup
	)

	go func() {
		for _, f := range files {
			filec <- f
		}
		close(filec)
	}()
	for i := runtime.NumCPU(); i >= 0; i-- {
		wg.Add(1)
		go getInfo(filec, infoc, &wg)
	}
	go func() {
		wg.Wait()
		close(infoc)
	}()
	writeLicenses(infoc)
}

func skipFile(path string) bool {
	if strings.Contains(path, "/testdata/") {
		return true
	}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func getFiles() []string {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
	var files []string
	err := doLines(cmd, func(line string) {
		if skipFile(line) {
			return
		}
		ext := filepath.Ext(line)
		for _, wantExt := range extensions {
			if ext == wantExt {
				goto keep
			}
		}
		return
	keep:
		files = append(files, line)
	})
	if err != nil {
		log.Fatal("error getting files:", err)
	}
	return files
}

func getInfo(files <-chan string, out chan<- *info, wg *sync.WaitGroup) {
	for file := range files {
		stat, err := os.Lstat(file)
		if err != nil {
			fmt.Printf("ERROR %s: %v\n", file, err)
			continue
		}
		if !stat.Mode().IsRegular() {
			continue
		}
		if isGenerated(file) {
			continue
		}
		out <- &info{file: file, Year: int64(time.Now().Year())}
	}
	wg.Done()
}

func isGenerated(file string) bool {
	fd, err := os.Open(file)
	if err != nil {
		return false
	}
	defer fd.Close()

	buf := make([]byte, 2048)
	n, _ := fd.Read(buf)
	buf = buf[:n]

	// Check if the file has a generated code or license header
	for _, l := range bytes.Split(buf, []byte("\n")) {
		if bytes.HasPrefix(l, []byte("// Code generated")) || bytes.HasPrefix(l, []byte("// Copyright")) {
			return true
		}
	}
	return false
}

func writeLicenses(infos <-chan *info) {
	for i := range infos {
		writeLicense(i)
	}
}

func writeLicense(info *info) {
	fi, err := os.Stat(info.file)
	if os.IsNotExist(err) {
		fmt.Println("skipping (does not exist)", info.file)
		return
	}
	if err != nil {
		log.Fatalf("error stat'ing %s: %v\n", info.file, err)
	}
	content, err := os.ReadFile(info.file)
	if err != nil {
		log.Fatalf("error reading %s: %v\n", info.file, err)
	}

	// Construct new file content.
	buf := new(bytes.Buffer)
	licenseT.Execute(buf, info)
	if m := licenseCommentRE.FindIndex(content); m != nil && m[0] == 0 {
		buf.Write(content[:m[0]])
		buf.Write(content[m[1]:])
	} else {
		buf.Write(content)
	}

	// Write it to the file.
	if bytes.Equal(content, buf.Bytes()) {
		fmt.Println("skipping (no changes)", info.file)
		return
	}
	fmt.Println("writing license to", info.file)
	if err := os.WriteFile(info.file, buf.Bytes(), fi.Mode()); err != nil {
		log.Fatalf("error writing %s: %v", info.file, err)
	}
}

func doLines(cmd *exec.Cmd, f func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s := bufio.NewScanner(stdout)
	for s.Scan() {
		f(s.Text())
	}
	if s.Err() != nil {
		return s.Err()
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%v (for %s)", err, strings.Join(cmd.Args, " "))
	}
	return nil
}
