// Copyright 2016-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// cp copies files.
//
// Synopsis:
//
//	cp [-rRfivwP] FROM... TO
//
// Options:
//
//	-w n: number of worker goroutines
//	-R: copy file hierarchies
//	-r: alias to -R recursive mode
//	-i: prompt about overwriting file
//	-f: force overwrite files
//	-v: verbose copy mode
//	-P: don't follow symlinks
package cp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"
)

type flags struct {
	recursive        bool
	ask              bool
	force            bool
	verbose          bool
	noFollowSymlinks bool
}

// promptOverwrite ask if the user wants overwrite file
func promptOverwrite(dst string, out io.Writer, in *bufio.Reader) (bool, error) {
	fmt.Fprintf(out, "cp: overwrite %q? ", dst)
	answer, err := in.ReadString('\n')
	if err != nil {
		return false, err
	}

	if strings.ToLower(answer)[0] != 'y' {
		return false, nil
	}

	return true, nil
}

func setupPreCallback(recursive, ask, force bool, writer io.Writer, reader bufio.Reader) func(string, string, os.FileInfo) error {
	return func(src, dst string, srcfi os.FileInfo) error {
		// check if src is dir
		if !recursive && srcfi.IsDir() {
			fmt.Fprintf(writer, "cp: -r not specified, omitting directory %s\n", src)
			return ErrSkip
		}

		dstfi, err := os.Stat(dst)
		if err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(writer, "cp: %q: can't handle error %v\n", dst, err)
			return ErrSkip
		} else if err != nil {
			// dst does not exist.
			return nil
		}

		// dst does exist.

		if os.SameFile(srcfi, dstfi) {
			fmt.Fprintf(writer, "cp: %q and %q are the same file\n", src, dst)
			return ErrSkip
		}
		if ask && !force {
			overwrite, err := promptOverwrite(dst, writer, &reader)
			if err != nil {
				return err
			}
			if !overwrite {
				return ErrSkip
			}
		}
		return nil
	}
}

func setupPostCallback(verbose bool, w io.Writer) func(src, dst string) {
	return func(src, dst string) {
		if verbose {
			fmt.Fprintf(w, "%q -> %q\n", src, dst)
		}
	}
}

// run evaluates the args and makes decisions for copyfiles
func run(params RunParams, args []string, f flags) error {
	todir := false
	from, to := args[:len(args)-1], args[len(args)-1]
	toStat, err := os.Stat(to)
	if err == nil {
		todir = toStat.IsDir()
	}
	if len(args) > 2 && !todir {
		return eNotDir
	}

	i := bufio.NewReader(params.Stdin)
	w := params.Stderr

	opts := Options{
		NoFollowSymlinks: f.noFollowSymlinks,

		// cp the command makes sure that
		//
		// (1) the files it's copying aren't already the same,
		// (2) the user is asked about overwriting an existing file if
		//     one is already there.
		PreCallback: setupPreCallback(f.recursive, f.ask, f.force, w, *i),

		PostCallback: setupPostCallback(f.verbose, w),
	}

	var lastErr error
	for _, file := range from {
		dst := to
		if todir {
			dst = filepath.Join(dst, filepath.Base(file))
		}
		absFile := params.MkAbs(file)
		absDst := params.MkAbs(dst)
		if f.recursive {
			lastErr = opts.CopyTree(absFile, absDst)
		} else {
			lastErr = opts.Copy(absFile, absDst)
		}
		var pathError *fs.PathError
		// Use the original path in errors.
		if errors.As(lastErr, &pathError) {
			switch pathError.Path {
			case absFile:
				pathError.Path = file
			case absDst:
				pathError.Path = dst
			}
		}
	}
	return lastErr
}

type RunParams struct {
	Dir string
	Env []string

	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

func (p RunParams) MkAbs(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(p.Dir, name)
}

func RunMain(params RunParams, args ...string) (exit int) {
	flagSet := flag.NewFlagSet("cp", flag.ContinueOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(params.Stderr, "Usage: cp [-wRrifvP] file[s] ... dest")
		flagSet.PrintDefaults()
	}
	var f flags
	flagSet.BoolVarP(&f.recursive, "RECURSIVE", "R", false, "copy file hierarchies")
	flagSet.BoolVarP(&f.recursive, "recursive", "r", false, "alias to -R recursive mode")
	flagSet.BoolVarP(&f.ask, "interactive", "i", false, "prompt about overwriting file")
	flagSet.BoolVarP(&f.force, "force", "f", false, "force overwrite files")
	flagSet.BoolVarP(&f.verbose, "verbose", "v", false, "verbose copy mode")
	flagSet.BoolVarP(&f.noFollowSymlinks, "no-dereference", "P", false, "don't follow symlinks")

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flagSet.NArg() < 2 {
		// TODO: print usage to the stderr parameter
		flagSet.Usage()
		return 0
	}

	if err := run(params, flagSet.Args(), f); err != nil {
		fmt.Fprintln(params.Stderr, err)
		return 1
	}
	return 0
}
