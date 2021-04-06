/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
)

// NextArg retrieves the next argument from the commandline.
func NextArg(i *int, args []string) string {
	(*i)++
	if (*i) >= len(args) {
		fmt.Fprintln(os.Stderr, "Expected another commandline argument.")
		os.Exit(1)
	}
	return args[*i]
}

// SimplifyArgs transforms an array of commandline arguments so that
// any -abc arg before the first -- (if any) are expanded into
// -a -b -c.
func SimplifyArgs(args []string) (r []string) {
	r = make([]string, 0, len(args)*2)
	for i, arg := range args {
		if arg == "--" {
			for j := i; j < len(args); j++ {
				r = append(r, args[j])
			}
			break
		}
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
			for j := 1; j < len(arg); j++ {
				r = append(r, "-"+string(arg[j]))
			}
		} else {
			r = append(r, arg)
		}
	}
	return
}

// SafeStrToInt returns the int or exits the process.
func SafeStrToInt(str string) (i int) {
	i, err := strconv.Atoi(str)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid integer \"%s\"\n", str)
		os.Exit(1)
	}
	return
}

// ReadInput gets Jsonnet code from the given place (file, commandline, stdin).
// It also updates the given filename to <stdin> or <cmdline> if it wasn't a
// real filename.
func ReadInput(filenameIsCode bool, filename *string) (input string, err error) {
	if filenameIsCode {
		input, err = *filename, nil
		*filename = "<cmdline>"
	} else if *filename == "-" {
		var bytes []byte
		bytes, err = ioutil.ReadAll(os.Stdin)
		input = string(bytes)
		*filename = "<stdin>"
	} else {
		var bytes []byte
		bytes, err = ioutil.ReadFile(*filename)
		input = string(bytes)
	}
	return
}

// SafeReadInput runs ReadInput, exiting the process if there was a problem.
func SafeReadInput(filenameIsCode bool, filename *string) string {
	output, err := ReadInput(filenameIsCode, filename)
	if err != nil {
		var op string
		switch typedErr := err.(type) {
		case *os.PathError:
			op = typedErr.Op
			err = typedErr.Err
		}
		if op == "open" {
			fmt.Fprintf(os.Stderr, "Opening input file: %s: %s\n", *filename, err.Error())
		} else if op == "read" {
			fmt.Fprintf(os.Stderr, "Reading input file: %s: %s\n", *filename, err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(1)
	}
	return output
}

// WriteOutputFile writes the output to the given file, creating directories
// if requested, and printing to stdout instead if the outputFile is "".
func WriteOutputFile(output string, outputFile string, createDirs bool) (err error) {
	if outputFile == "" {
		fmt.Print(output)
		return nil
	}

	if createDirs {
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return err
		}
	}

	f, createErr := os.Create(outputFile)
	if createErr != nil {
		return createErr
	}
	defer func() {
		if ferr := f.Close(); ferr != nil {
			err = ferr
		}
	}()

	_, err = f.WriteString(output)
	return err
}

// StartCPUProfile creates a CPU profile if requested by environment
// variable.
func StartCPUProfile() {
	// https://blog.golang.org/profiling-go-programs
	var cpuprofile = os.Getenv("JSONNET_CPU_PROFILE")
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// StopCPUProfile ensures any running CPU profile is stopped.
func StopCPUProfile() {
	var cpuprofile = os.Getenv("JSONNET_CPU_PROFILE")
	if cpuprofile != "" {
		pprof.StopCPUProfile()
	}
}

// MemProfile creates a memory profile if requested by environment
// variable.
func MemProfile() {
	var memprofile = os.Getenv("JSONNET_MEM_PROFILE")
	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Fatal("Failed to close the memprofile: ", err)
			}
		}()
	}
}
