/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package copyright

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/blend/go-sdk/stringutil"
)

// New creates a new profanity engine with a given set of config options.
func New(options ...Option) *Copyright {
	var c Copyright
	for _, option := range options {
		option(&c)
	}
	return &c
}

// Copyright is the main type that injects, removes and verifies copyright headers.
type Copyright struct {
	Config // Config holds the configuration opitons.

	// Stdout is the writer for Verbose and Debug output.
	// If it is unset, `os.Stdout` will be used.
	Stdout io.Writer
	// Stderr is the writer for Error output.
	// If it is unset, `os.Stderr` will be used.
	Stderr io.Writer
}

// Inject inserts the copyright header in any matching files that don't already
// have the copyright header.
func (c Copyright) Inject(ctx context.Context) error {
	return c.Walk(ctx, c.inject)
}

// Remove removes the copyright header in any matching files that
// have the copyright header.
func (c Copyright) Remove(ctx context.Context) error {
	return c.Walk(ctx, c.remove)
}

// Verify asserts that the files found during walk
// have the copyright header.
func (c Copyright) Verify(ctx context.Context) error {
	return c.Walk(ctx, c.verify)
}

// Walk traverses the tree recursively from "." and applies the given action.
func (c Copyright) Walk(ctx context.Context, action Action) error {
	notice, err := c.compileNoticeBodyTemplate(c.NoticeBodyTemplateOrDefault())
	if err != nil {
		return err
	}
	c.Verbosef("using include files: %s", strings.Join(c.IncludeFiles, ", "))
	c.Verbosef("using include dirs: %s", strings.Join(c.IncludeDirs, ", "))
	c.Verbosef("using exclude files: %s", strings.Join(c.ExcludeFiles, ", "))
	c.Verbosef("using exclude dirs: %s", strings.Join(c.ExcludeDirs, ", "))
	c.Verbosef("using notice:\n%s", c.prefix("\t", strings.TrimSpace(notice)))

	var didFail bool
	err = filepath.Walk(c.RootOrDefault(), func(path string, info os.FileInfo, fileErr error) error {
		if fileErr != nil {
			return fileErr
		}

		if info.IsDir() {
			if path == c.RootOrDefault() {
				return nil
			}

			for _, exclude := range c.ExcludeDirsOrDefault() {
				if stringutil.Glob(path, exclude) {
					c.Debugf("%s: skipping dir (matches exclude glob: %s)", path, exclude)
					return filepath.SkipDir
				}
			}

			var includeDir bool
			for _, include := range c.IncludeDirsOrDefault() {
				if stringutil.Glob(path, include) {
					includeDir = true
					break
				}
			}
			if !includeDir {
				c.Debugf("%s: skipping dir (doesnt match any include globs: %s)", path, strings.Join(c.IncludeDirsOrDefault(), ", "))
				return filepath.SkipDir
			}
			return nil
		}

		for _, exclude := range c.ExcludeFilesOrDefault() {
			if stringutil.Glob(path, exclude) {
				c.Debugf("%s: skipping file (matches exclude glob: %s)", path, exclude)
				return nil
			}
		}

		var includeFile bool
		for _, include := range c.IncludeFilesOrDefault() {
			if stringutil.Glob(path, include) {
				includeFile = true
				break
			}
		}
		if !includeFile {
			c.Debugf("%s: skipping file (doesnt match any include globs: %s)", path, strings.Join(c.IncludeFilesOrDefault(), ", "))
			return nil
		}

		fileExtension := filepath.Ext(path)

		// test the file
		noticeTemplate, ok := c.NoticeTemplatesOrDefault()[fileExtension]
		if !ok {
			return fmt.Errorf("invalid copyright injection file; %s", fileExtension)
		}

		noticeCompiled, err := c.compileNoticeTemplate(noticeTemplate, notice)
		if err != nil {
			return err
		}

		fileContents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		noticeCompiledBytes := []byte(noticeCompiled)
		err = action(path, info, fileContents, noticeCompiledBytes)
		if err != nil {
			if err == ErrFailure {
				didFail = true
				return nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if didFail {
		return ErrFailure
	}
	return nil
}

func (c Copyright) inject(path string, info os.FileInfo, file, notice []byte) error {
	injectedContents := c.injectedContents(path, file, notice)
	if injectedContents == nil {
		return nil
	}
	return ioutil.WriteFile(path, injectedContents, info.Mode().Perm())
}

func (c Copyright) injectedContents(path string, file, notice []byte) []byte {
	fileExtension := filepath.Ext(path)
	var injectedContents []byte
	if fileExtension == ".go" {
		injectedContents = c.goInjectNotice(path, file, notice)
	} else {
		injectedContents = c.injectNotice(path, file, notice)
	}
	return injectedContents
}

func (c Copyright) remove(path string, info os.FileInfo, file, notice []byte) error {
	removedContents := c.removedContents(path, file, notice)
	if removedContents == nil {
		return nil
	}
	return ioutil.WriteFile(path, removedContents, info.Mode().Perm())
}

func (c Copyright) removedContents(path string, file, notice []byte) []byte {
	fileExtension := filepath.Ext(path)
	var removedContents []byte
	if fileExtension == ".go" {
		removedContents = c.goRemoveNotice(path, file, notice)
	} else {
		removedContents = c.removeNotice(path, file, notice)
	}
	return removedContents
}

func (c Copyright) verify(path string, _ os.FileInfo, file, notice []byte) error {
	fileExtension := filepath.Ext(path)
	var err error
	if fileExtension == ".go" {
		err = c.goVerifyNotice(path, file, notice)
	} else {
		err = c.verifyNotice(path, file, notice)
	}
	if c.Config.ExitFirstOrDefault() {
		return err
	}
	if err != nil {
		fmt.Fprintf(c.GetStderr(), "%+v\n", err)
		return ErrFailure
	}
	return nil
}

// GetStdout returns standard out.
func (c Copyright) GetStdout() io.Writer {
	if c.QuietOrDefault() {
		return ioutil.Discard
	}
	if c.Stdout != nil {
		return c.Stdout
	}
	return os.Stdout
}

// GetStderr returns standard error.
func (c Copyright) GetStderr() io.Writer {
	if c.QuietOrDefault() {
		return ioutil.Discard
	}
	if c.Stderr != nil {
		return c.Stderr
	}
	return os.Stderr
}

// Verbosef writes to stdout if the `Verbose` flag is true.
func (c Copyright) Verbosef(format string, args ...interface{}) {
	if !c.VerboseOrDefault() {
		return
	}
	fmt.Fprintf(c.GetStdout(), format+"\n", args...)
}

// Debugf writes to stdout if the `Debug` flag is true.
func (c Copyright) Debugf(format string, args ...interface{}) {
	if !c.DebugOrDefault() {
		return
	}
	fmt.Fprintf(c.GetStdout(), format+"\n", args...)
}

//
// internal helpers
//

// goInjectNotice handles go files differently because they may contain build tags.
func (c Copyright) goInjectNotice(path string, file, notice []byte) []byte {
	goBuildTag := goBuildTagMatch.FindString(string(file))
	file = goBuildTagMatch.ReplaceAll(file, nil)
	if c.fileHasCopyrightHeader(file, notice) {
		return nil
	}

	c.Verbosef("injecting notice: %s", path)
	return c.mergeFileSections([]byte(goBuildTag), notice, file)
}

func (c Copyright) injectNotice(path string, file, notice []byte) []byte {
	if c.fileHasCopyrightHeader(file, notice) {
		return nil
	}
	c.Verbosef("injecting notice: %s", path)
	return c.mergeFileSections(notice, file)
}

func (c Copyright) goRemoveNotice(path string, file, notice []byte) []byte {
	goBuildTag := goBuildTagMatch.FindString(string(file))
	file = goBuildTagMatch.ReplaceAll(file, []byte(""))
	if !c.fileHasCopyrightHeader(file, notice) {
		return nil
	}
	c.Verbosef("removing notice: %s", path)
	return c.mergeFileSections([]byte(goBuildTag), c.removeCopyrightHeader(file, notice))
}

func (c Copyright) removeNotice(path string, file, notice []byte) []byte {
	if !c.fileHasCopyrightHeader(file, notice) {
		return nil
	}
	c.Verbosef("removing notice: %s", path)
	return c.removeCopyrightHeader(file, notice)
}

func (c Copyright) goVerifyNotice(path string, file, notice []byte) error {
	c.Debugf("verifying notice: %s", path)
	file = goBuildTagMatch.ReplaceAll(file, nil)
	if !c.fileHasCopyrightHeader(file, notice) {
		return fmt.Errorf(verifyErrorFormat, path)
	}
	return nil
}

func (c Copyright) verifyNotice(path string, file, notice []byte) error {
	c.Debugf("verifying notice: %s", path)
	if !c.fileHasCopyrightHeader(file, notice) {
		return fmt.Errorf(verifyErrorFormat, path)
	}
	return nil
}

func (c Copyright) fileHasCopyrightHeader(fileContents, notice []byte) bool {
	fileContentsString := string(fileContents)
	noticeMatch := c.createNoticeMatchExpression(notice)
	return noticeMatch.MatchString(fileContentsString)
}

func (c Copyright) removeCopyrightHeader(fileContents, notice []byte) []byte {
	fileContentsString := string(fileContents)
	noticeMatch := c.createNoticeMatchExpression(notice)
	return []byte(noticeMatch.ReplaceAllString(fileContentsString, ""))
}

func (c Copyright) createNoticeMatchExpression(notice []byte) *regexp.Regexp {
	noticeString := string(notice)
	noticeExpr := yearMatch.ReplaceAllString(regexp.QuoteMeta(noticeString), yearExpr)
	noticeExpr = `^(\s*)` + noticeExpr
	return regexp.MustCompile(noticeExpr)
}

func (c Copyright) mergeFileSections(sections ...[]byte) []byte {
	var fullLength int
	for _, section := range sections {
		fullLength += len(section)
	}

	combined := make([]byte, fullLength)

	var written int
	for _, section := range sections {
		copy(combined[written:], section)
		written += len(section)
	}
	return combined
}

func (c Copyright) prefix(prefix string, s string) string {
	lines := strings.Split(s, "\n")
	var output []string
	for _, l := range lines {
		output = append(output, prefix+l)
	}
	return strings.Join(output, "\n")
}

func (c Copyright) compileNoticeTemplate(noticeTemplate, notice string) (string, error) {
	return c.processTemplate(noticeTemplate, c.templateViewModel(map[string]interface{}{
		"Notice": notice,
	}))
}

func (c Copyright) templateViewModel(extra ...map[string]interface{}) map[string]interface{} {
	base := map[string]interface{}{
		"Year":    c.YearOrDefault(),
		"Company": c.CompanyOrDefault(),
		"License": c.LicenseOrDefault(),
	}
	for _, m := range extra {
		for key, value := range m {
			base[key] = value
		}
	}
	return base
}

func (c Copyright) compileRestrictionsTemplate(restrictionsTemplate string) (string, error) {
	return c.processTemplate(restrictionsTemplate, c.templateViewModel())
}

func (c Copyright) compileNoticeBodyTemplate(noticeBodyTemplate string) (string, error) {
	restrictions, err := c.compileRestrictionsTemplate(c.RestrictionsOrDefault())
	if err != nil {
		return "", err
	}
	viewModel := c.templateViewModel(map[string]interface{}{
		"Restrictions": restrictions,
	})
	output, err := c.processTemplate(noticeBodyTemplate, viewModel)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (c Copyright) processTemplate(text string, viewmodel interface{}) (string, error) {
	tmpl := template.New("output")
	tmpl = tmpl.Funcs(template.FuncMap{
		"prefix": c.prefix,
	})
	compiled, err := tmpl.Parse(text)
	if err != nil {
		return "", err
	}

	output := new(bytes.Buffer)
	if err = compiled.Execute(output, viewmodel); err != nil {
		return "", err
	}
	return output.String(), nil
}
