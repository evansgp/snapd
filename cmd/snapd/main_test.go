// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/client"
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/testutil"

	snapd "github.com/snapcore/snapd/cmd/snapd"
)

// Hook up check.v1 into the "go test" runner
func Test(t *testing.T) { TestingT(t) }

type snapdSuite struct {
	tmpdir string
}

var _ = Suite(&snapdSuite{})

func (s *snapdSuite) SetUpTest(c *C) {
	s.tmpdir = c.MkDir()
	for _, d := range []string{"/var/lib/snapd", "/run"} {
		err := os.MkdirAll(filepath.Join(s.tmpdir, d), 0755)
		c.Assert(err, IsNil)
	}
	dirs.SetRootDir(s.tmpdir)
}

func (s *snapdSuite) TestSelftestFailGoesIntoDegradedMode(c *C) {
	logbuf, restore := logger.MockLogger()
	defer restore()

	selftestErr := fmt.Errorf("foo failed")
	selftestWasRun := 0
	restore = snapd.MockSelftestRun(func() error {
		selftestWasRun += 1
		return selftestErr
	})
	defer restore()

	restore = snapd.MockCheckRunningConditionsRetryDelay(10 * time.Millisecond)
	defer restore()

	// run the daemon
	ch := make(chan os.Signal)
	go func() {
		err := snapd.Run(ch)
		c.Assert(err, IsNil)
	}()
	time.Sleep(100 * time.Millisecond)

	// verify that talking to the daemon yields the selftest error
	// message
	cli := client.New(nil)
	_, err := cli.Abort("123")
	c.Check(selftestWasRun >= 1, Equals, true)
	c.Check(err, ErrorMatches, "system does not fully support snapd: foo failed")
	c.Check(logbuf.String(), testutil.Contains, "system does not fully support snapd: foo failed")

	// verify that the sysinfo command is still available
	_, err = cli.SysInfo()
	c.Check(err, IsNil)

	// stop the daemon
	close(ch)
}
