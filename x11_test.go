package main

import (
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/notedit/gst"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type Xvfb struct {
	conn    *xgb.Conn
	Display string
	Width   int
	Height  int
	Depth   int
}

func NewXvfb(t *testing.T, display string, width, height, depth int, timeout time.Duration) *Xvfb {
	cmd := exec.Command("Xvfb", display, "-screen", "0", fmt.Sprintf("%dx%dx%d", width, height, depth))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	connch := make(chan *xgb.Conn)
	errch := make(chan error)
	go func() {
		for {
			conn, err := xgb.NewConnDisplay(display)
			if err != nil {
				if strings.Contains(err.Error(), "cannot connect to ") {
					continue
				}
				errch <- err
			}
			connch <- conn
		}
	}()
	select {
	case conn := <-connch:
		return &Xvfb{
			conn:    conn,
			Display: display,
			Width:   width,
			Height:  height,
			Depth:   depth,
		}
	case err := <-errch:
		t.Fatalf("failed to connect to Xvfb display %s %+v", display, err)
	case <-time.After(timeout):
		t.Fatalf("failed to connect to Xvfb display %s timeout", display)
	}
	return nil
}

func (x *Xvfb) StartCommand(t *testing.T, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{fmt.Sprintf("DISPLAY=%s", x.Display)}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	return cmd
}

func startPipeline(t *testing.T, pipelineStr string) *gst.Pipeline {
	p, err := gst.ParseLaunch(pipelineStr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		p.SetState(gst.StateNull)
	})
	p.SetState(gst.StatePlaying)
	return p
}

func pullRGBAImage(t *testing.T, e *gst.Element, width, height int) *image.RGBA {
	sample, err := e.PullSample()
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Pix = sample.Data
	return img
}

func savePNG(t *testing.T, filename string, img image.Image) {
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestXImageSrc(t *testing.T) {
	display := os.Getenv("DISPLAY")
	width, height := 100, 100

	cmd := exec.Command("wine", "notepad")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	time.Sleep(10 * time.Second)

	ps1 := fmt.Sprintf(`ximagesrc display-name=%s use-damage=0 endx=%d endy=%d
! videoconvert
! pngenc
! appsink name=dst drop=1`, display, width-1, height-1)
	log.Printf("%s", ps1)

	p1, err := gst.ParseLaunch(ps1)
	if err != nil {
		log.Fatal(err)
	}
	dst1 := p1.GetByName("dst")
	p1.SetState(gst.StatePlaying)

	sample, err := dst1.PullSample()
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile("png.png", sample.Data, 0775)
}
