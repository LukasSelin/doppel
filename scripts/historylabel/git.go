package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// repo reads objects out of one git repository through a single long-lived
// `git cat-file --batch`, because a history walk asks for tens of thousands of
// trees and blobs and a process per object is minutes on Windows.
type repo struct {
	dir string
	in  io.WriteCloser
	out *bufio.Reader
	cmd *exec.Cmd
}

func openRepo(dir string) (*repo, error) {
	cmd := exec.Command("git", "-C", dir, "cat-file", "--batch")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &repo{dir: dir, in: in, out: bufio.NewReaderSize(out, 1<<20), cmd: cmd}, nil
}

func (r *repo) close() {
	r.in.Close()
	r.cmd.Wait()
}

// object returns an object's sha, type and content; ok is false when the spec
// names nothing (a path that does not exist at that revision).
func (r *repo) object(spec string) (sha, typ string, data []byte, ok bool, err error) {
	if _, err = fmt.Fprintln(r.in, spec); err != nil {
		return
	}
	header, err := r.out.ReadString('\n')
	if err != nil {
		return
	}
	f := strings.Fields(header)
	if len(f) == 2 && f[1] == "missing" || len(f) != 3 {
		return "", "", nil, false, nil
	}
	size, err := strconv.Atoi(f[2])
	if err != nil {
		return
	}
	data = make([]byte, size+1) // content plus the trailing newline
	if _, err = io.ReadFull(r.out, data); err != nil {
		return
	}
	return f[0], f[1], data[:size], true, nil
}

// treeEntry is one entry of a git tree object.
type treeEntry struct {
	mode, name, sha string
}

func parseTree(data []byte) []treeEntry {
	var out []treeEntry
	for len(data) > 0 {
		sp := bytes.IndexByte(data, ' ')
		nul := bytes.IndexByte(data, 0)
		if sp < 0 || nul < 0 || nul+21 > len(data) {
			break
		}
		out = append(out, treeEntry{
			mode: string(data[:sp]),
			name: string(data[sp+1 : nul]),
			sha:  hex.EncodeToString(data[nul+1 : nul+21]),
		})
		data = data[nul+21:]
	}
	return out
}

// git runs one plain git command and returns its stdout.
func (r *repo) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}
