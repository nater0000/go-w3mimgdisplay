/*
 * Copyright (c) 2019 Paul Seyfert
 * Author: Paul Seyfert <pseyfert.mathphys@gmail.com>
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *     * Redistributions of source code must retain the above copyright
 *       notice, this list of conditions and the following disclaimer.
 *     * Redistributions in binary form must reproduce the above copyright
 *       notice, this list of conditions and the following disclaimer in the
 *       documentation and/or other materials provided with the distribution.
 *     * Neither the name of the <organization> nor the
 *       names of its contributors may be used to endorse or promote products
 *       derived from this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package w3mimgdisplay

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/renameio"
)

func PrintImage(x, y int, img image.Image) error {
	if globals.w3m_proc == nil {
		render_init()
	} else {
		render_step()
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)

	t, err := renameio.TempFile(globals.tmpdir, globals.file)
	if err != nil {
		return fmt.Errorf("Cannot create temporary image file: %v", err)
	}
	defer t.Cleanup()

	_, err = t.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("Cannot write to temporary image file: %v", err)
	}
	err = t.CloseAtomicallyReplace()
	if err != nil {
		return fmt.Errorf("Cannot finalize to temporary image file: %v", err)
	}

	b := img.Bounds()
	windowWidth := b.Dx()
	windowHeight := b.Dy()

	// io.WriteString(globals.w3m_pipe, fmt.Sprintf("0;1;%d;%d;%d;%d;;;;;%s\n4;\n3;\n", x, y, windowWidth, windowHeight, globals.file))
	io.WriteString(globals.w3m_pipe, fmt.Sprintf("0;1;%d;%d;%d;%d;%d;%d;%d;%d;%s\n4;\n3;\n", x, y, windowWidth, windowHeight, b.Min.X, b.Min.Y, windowWidth, windowHeight, globals.file))

	return nil
}

var globals struct {
	tmpdir    string
	file      string
	id        int
	w3m_proc  *exec.Cmd
	w3m_pipe  io.WriteCloser
	cleanchan chan int
}

func render_init() error {
	var err error
	globals.w3m_proc = exec.Command("/usr/lib/w3m/w3mimgdisplay")
	globals.w3m_pipe, err = globals.w3m_proc.StdinPipe()
	if err != nil {
		return fmt.Errorf("Cannot open stdin to w3mimgdisplay: %v", err)
	}
	globals.w3m_proc.Start()
	globals.tmpdir, err = ioutil.TempDir("", "invaders")
	if err != nil {
		return fmt.Errorf("Cannot create temporary directory: %v", err)
	}
	globals.cleanchan = make(chan int, 50) // backlog of no more than 50 undeleted images
	render_step()
	go concurrent_clean()
	return nil
}

func render_step() {
	goodToDelete := globals.id
	globals.id += 1
	globals.file = filepath.Join(globals.tmpdir, fmt.Sprintf("buffer%d.png", globals.id))
	globals.cleanchan <- goodToDelete
}

func concurrent_clean() {
	for {
		rmid, ok := <-globals.cleanchan
		if !ok {
			break
		}
		os.Remove(filepath.Join(globals.tmpdir, fmt.Sprintf("buffer%d.png", rmid)))
	}
}

func render_cleanup() {
	close(globals.cleanchan)
	globals.w3m_pipe.Close()
	globals.w3m_proc.Wait()
	os.RemoveAll(globals.tmpdir)
}
