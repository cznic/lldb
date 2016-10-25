// Copyright 2014 The lldb Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lldb

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cznic/strutil"
)

const (
	testDbName = "_test.db"
	walName    = "_wal"
)

func caller(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(2)
	fmt.Fprintf(os.Stderr, "caller: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	_, fn, fl, _ = runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "\tcallee: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintln(os.Stderr)
	_ = os.Stderr.Sync()
}

func dbg(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "dbg %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	_ = os.Stderr.Sync()
}

func TODO(...interface{}) string {
	_, fn, fl, _ := runtime.Caller(1)
	return fmt.Sprintf("TODO: %s:%d:\n", path.Base(fn), fl)
}

func use(...interface{}) {}

// ============================================================================

func now() time.Time { return time.Now() }

func hdump(b []byte) string {
	return hex.Dump(b)
}

func die() {
	os.Exit(1)
}

func stack() string {
	buf := make([]byte, 1<<16)
	return string(buf[:runtime.Stack(buf, false)])
}

func temp() (dir, name string) {
	dir, err := ioutil.TempDir("", "test-lldb-")
	if err != nil {
		panic(err)
	}

	return dir, filepath.Join(dir, "test.tmp")
}

func testIssue12(t *testing.T, keys []int, compress bool) {
	dir, fn := temp()
	defer os.RemoveAll(dir)

	f, err := os.Create(fn)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	fil := NewSimpleFileFiler(f)
	a, err := NewAllocator(fil, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	a.Compress = compress
	t.Logf("Using compression (recommended): %v", compress)
	btree, _, err := CreateBTree(a, bytes.Compare)
	if err != nil {
		t.Fatal(err)
	}

	t0 := time.Now()
	for _, i := range keys {
		k := make([]byte, 4)
		binary.BigEndian.PutUint32(k, uint32(i))
		if err = btree.Set(k, nil); err != nil {
			t.Fatal(err)
		}
	}

	d := time.Since(t0)
	t.Logf("%d keys in %s, %f keys/s, %v s/key", len(keys), d, float64(len(keys))/d.Seconds(), d/time.Duration(len(keys)))

	sz, err := fil.Size()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("File size: %d, %f bytes/key", sz, float64(sz)/float64(len(keys)))

	var stats AllocStats
	if err := a.Verify(NewMemFiler(), nil, &stats); err != nil {
		t.Fatal(err)
	}

	t.Logf("\n%s", strutil.PrettyString(stats, "", "", nil))
}

func TestIssue12(t *testing.T) {
	fmt.Fprintf(os.Stderr, "TestIssue12: Warning, run with -timeout at least 1h\n")
	keys := rand.Perm(1 << 20)
	testIssue12(t, keys, false)
	testIssue12(t, keys, true)
}
