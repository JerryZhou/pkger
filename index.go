package pkger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobuffalo/here"
)

type index struct {
	Files map[Path]*File
}

func (i *index) Create(pt Path) (*File, error) {
	her, err := Info(pt.Pkg)
	if err != nil {
		return nil, err
	}
	f := &File{
		path: pt,
		her:  her,
		info: &FileInfo{
			name:    strings.TrimPrefix(pt.Name, "/"),
			mode:    0666,
			modTime: time.Now(),
		},
	}

	i.Files[pt] = f
	return f, nil
}

func (i index) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}

	fm := map[string]File{}

	for k, v := range i.Files {
		fm[k.String()] = *v
	}

	m["files"] = fm

	return json.Marshal(m)
}

func (i index) Walk(pt Path, wf WalkFunc) error {
	if len(i.Files) > 0 {
		for k, v := range i.Files {
			if k.Pkg != pt.Pkg {
				continue
			}
			if err := wf(k, v.info); err != nil {
				return err
			}
		}
	}

	var info here.Info
	var err error
	if pt.Pkg == "." {
		info, err = Current()
		if err != nil {
			return err
		}
		pt.Pkg = info.ImportPath
	}

	if info.IsZero() {
		info, err = Info(pt.Pkg)
		if err != nil {
			return fmt.Errorf("%s: %s", pt, err)
		}
	}
	fp := filepath.Join(info.Dir, pt.Name)
	err = filepath.Walk(fp, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		path = strings.TrimPrefix(path, info.Dir)
		pt, err := Parse(fmt.Sprintf("%s:%s", pt.Pkg, path))
		if err != nil {
			return err
		}
		return wf(pt, NewFileInfo(fi))
	})

	return err
}

func (i *index) Open(pt Path) (*File, error) {
	f, ok := i.Files[pt]
	if !ok {
		return i.openDisk(pt)
	}
	nf := &File{
		info: f.info,
		path: f.path,
		data: f.data,
		her:  f.her,
	}

	return nf, nil
}

func (i index) openDisk(pt Path) (*File, error) {
	info, err := Info(pt.Pkg)
	if err != nil {
		return nil, err
	}
	fp := info.Dir
	if len(pt.Name) > 0 {
		fp = filepath.Join(fp, pt.Name)
	}

	fi, err := os.Stat(fp)
	if err != nil {
		return nil, err
	}
	f := &File{
		info: WithName(strings.TrimPrefix(pt.Name, "/"), NewFileInfo(fi)),
		her:  info,
		path: pt,
	}
	return f, nil
}

func newIndex() *index {
	return &index{
		Files: map[Path]*File{},
	}
}

var rootIndex = func() *index {
	i := newIndex()
	return i
}()
