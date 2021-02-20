package gst_x11

import (
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/notedit/gst"
	"github.com/stretchr/testify/assert"
	"image"
	"image/png"
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
	var sample *gst.Sample
	for {
		s, err := e.PullSample()
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range s.Data {
			if b != 0 {
				sample = s
				goto OK
			}
		}
	}
OK:
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
	display := ":99"
	width, height := 50, 50
	xvfb := NewXvfb(t, display, width, height, 24, 5*time.Second)
	xvfb.StartCommand(t, "xeyes", "-geometry", fmt.Sprintf("%dx%d", width, height))

	{
		ps1 := fmt.Sprintf(`ximagesrc name=src display-name=%s show-pointer=0 use-damage=0
! videoconvert
! video/x-raw,format=RGBA
! appsink name=dst`, display)

		p1 := startPipeline(t, ps1)
		dst1 := p1.GetByName("dst")
		assert.NotNil(t, dst1)
		img1 := pullRGBAImage(t, dst1, width, height)
		savePNG(t, "image1.png", img1)
	}

	{
		ps2 := fmt.Sprintf(`ximagesrc name=src display-name=%s show-pointer=0 use-damage=0 endx=29 endy=29
! videoconvert
! video/x-raw,format=RGBA
! appsink name=dst`, display)
		p2 := startPipeline(t, ps2)
		dst2 := p2.GetByName("dst")
		assert.NotNil(t, dst2)
		img2 := pullRGBAImage(t, dst2, 30, 30)
		savePNG(t, "image2.png", img2)
	}
}
