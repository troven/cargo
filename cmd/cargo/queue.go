package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	"github.com/xlab/treeprint"
)

type Queue []QueueAction

func NewQueue(actions ...QueueAction) Queue {
	return Queue(actions)
}

func (q Queue) Description(title string) string {
	if len(title) == 0 {
		title = "Queue"
	}
	t := treeprint.New()
	t = t.AddBranch(title)
	for i, action := range q {
		t.AddMetaNode(fmt.Sprintf("%d", i+1), action.Comment())
	}
	return t.String()
}

func (q Queue) Exec() bool {
	qq := make(Queue, 0, len(q))
	revertPrevious := func(qq Queue) {
		for i := len(qq) - 1; i >= 0; i-- {
			if err := qq[i].Revert(); err != nil {
				log.Errorf("revert Action#%d failed: %v", i+1, err)
			} else {
				log.Warningf("reverted Action#%d", i+1)
			}
		}
	}
	for i, action := range q {
		log.Infof("action#%d: %s", i+1, action.Comment())
		f, err := action.Run()
		if err != nil {
			log.Errorf("action#%d error: %v", i+1, err)
			revertPrevious(qq)
			return false
		}
		qq = append(qq, action)
		if err := action.Finalize(f); err != nil {
			log.Errorf("finalizer#%d error: %v", i+1, err)
			revertPrevious(qq)
			return false
		}
	}
	return true
}

type QueueAction interface {
	Run() (*os.File, error)
	Comment() string
	Finalize(f *os.File) error
	Revert() error
}

func CheckDirAction(dstDir, path string) QueueAction {
	return &queueAction{
		action: func() (*os.File, error) {
			info, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			if !info.IsDir() {
				return nil, errors.New(path + " is not a dir")
			}
			return nil, nil
		},
		comment: fmt.Sprintf("dir %s must exist", dstPath(dstDir, path)),
	}
}

func NewDirAction(dstDir, path string) QueueAction {
	return &queueAction{
		action: func() (*os.File, error) {
			err := os.MkdirAll(path, 0755)
			return nil, err
		},
		comment: fmt.Sprintf("new dir %s if not exists", dstPath(dstDir, path)),
		revert: func() error {
			return os.Remove(path)
		},
	}
}

func mkDirFor(path string) error {
	targetDir := filepath.Dir(path)
	if info, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err = os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
	} else if !info.IsDir() {
		err := fmt.Errorf("target directory is not a directory: %s", targetDir)
		return err
	}
	return nil
}

func CreateNewFileAction(dstDir, path string, contents []byte) QueueAction {
	return &queueAction{
		action: func() (f *os.File, err error) {
			if err := mkDirFor(path); err != nil {
				return nil, err
			}
			return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		},
		comment: fmt.Sprintf("new file %s size=%s (no overwrite)",
			dstPath(dstDir, path), contentSize(contents)),
		finalize: func(f *os.File) error {
			if f == nil {
				return nil
			}
			defer f.Close()
			return flushBufferToFile(contents, f)
		},
		revert: func() error {
			return os.Remove(path)
		},
	}
}

func OverwriteFileAction(dstDir, path string, contents []byte) QueueAction {
	return &queueAction{
		action: func() (f *os.File, err error) {
			if err := mkDirFor(path); err != nil {
				return nil, err
			}
			return os.Create(path)
		},
		comment: fmt.Sprintf("overwrite file %s size=%s",
			dstPath(dstDir, path), contentSize(contents)),
		finalize: func(f *os.File) error {
			if f == nil {
				return nil
			}
			defer f.Close()
			return flushBufferToFile(contents, f)
		},
		revert: func() error {
			return os.Remove(path)
		},
	}
}

func CopyFileAction(dstDir, dst, src string) QueueAction {
	return &queueAction{
		action: func() (f *os.File, err error) {
			if err := mkDirFor(dst); err != nil {
				return nil, err
			}
			return os.Create(dst)
		},
		comment: fmt.Sprintf("copy file %s", dstPath(dstDir, dst)),
		finalize: func(dstFile *os.File) error {
			if dstFile == nil {
				return nil
			}
			defer dstFile.Close()
			srcFile, err := os.Open(src)
			if err != nil {
				return err
			}
			return copyFileToFile(dstFile, srcFile)
		},
		revert: func() error {
			return os.Remove(dst)
		},
	}
}

type queueAction struct {
	action   func() (*os.File, error)
	comment  string
	finalize func(f *os.File) error
	revert   func() error
}

func (q *queueAction) Run() (*os.File, error) {
	if q.action != nil {
		return q.action()
	}
	return nil, nil
}

func (q *queueAction) Comment() string {
	return q.comment
}

func (q *queueAction) Finalize(f *os.File) error {
	if q.finalize != nil {
		return q.finalize(f)
	}
	return nil
}

func (q *queueAction) Revert() error {
	if q.revert != nil {
		return q.revert()
	}
	return nil
}

func contentSize(contents []byte) string {
	return humanize.Bytes(uint64(len(contents)))
}

func dstPath(dstDir, path string) string {
	return filepath.Join("[dst]", strings.TrimPrefix(path, dstDir))
}

func copyFileToFile(dst, src *os.File) error {
	_, err := io.Copy(dst, src)
	return err
}

func flushBufferToFile(buf []byte, f *os.File) error {
	_, err := io.Copy(f, bytes.NewReader(buf))
	return err
}
