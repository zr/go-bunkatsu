package downloader

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
)

// Downloader ダウンロードに関する全体的なロジックの型
type Downloader struct {
	url           string
	name          string
	tmpDir        string
	contentLength int64
	timeout       time.Duration
	division      int
	processList   *[]process
}

type process struct {
	index     int
	filePath  string
	startByte int64
	endByte   int64
}

const (
	timeout  = 10
	division = 5
)

var (
	errTooFewArgument = errors.New("Too Few Arguments")
	errContentLength  = errors.New("Invalid Content Length")
	errAcceptRanges   = errors.New("Range Access Is Not Accepted")
	errInterrupt      = errors.New("Interruption Detected")
)

// Run 指定したURLからファイルをダウンロードする
func Run() error {
	url, err := parseURL()
	if err != nil {
		return err
	}

	d, err := newDownloader(url)
	if err != nil {
		return err
	}
	defer os.RemoveAll(d.tmpDir)

	done := make(chan error, 1)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)

	go d.execute(done)

	select {
	case err, ok := <-done:
		if ok {
			return err
		}
		return nil
	case <-quit:
		return errInterrupt
	}

}

func parseURL() (string, error) {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return "", errTooFewArgument
	}

	return args[0], nil
}

func newDownloader(url string) (*Downloader, error) {
	name := filepath.Base(url)

	dir, err := ioutil.TempDir("", "dlr")
	if err != nil {
		return &Downloader{}, err
	}

	res, err := http.Head(url)
	if err != nil {
		return &Downloader{}, err
	}

	ars := res.Header["Accept-Ranges"]
	if len(ars) == 0 {
		return &Downloader{}, errAcceptRanges
	}

	clg := res.ContentLength
	if clg == -1 {
		return &Downloader{}, errContentLength
	}

	return &Downloader{
		url:           url,
		name:          name,
		tmpDir:        dir,
		contentLength: clg,
		timeout:       timeout,
		division:      division,
	}, nil
}

func (d *Downloader) execute(done chan<- error) {
	defer close(done)
	if err := d.createProcessList(); err != nil {
		done <- err
	}

	if err := d.download(); err != nil {
		done <- err
	}

	if err := d.merge(); err != nil {
		done <- err
	}

	done <- nil
}

func (d *Downloader) createProcessList() error {

	processList := []process{}

	rangeSize := d.contentLength / int64(d.division)

	for i := 0; i < d.division; i++ {
		startByte := rangeSize * int64(i)
		endByte := startByte + rangeSize - int64(1)

		if i == d.division-1 {
			endByte = d.contentLength
		}

		process := process{
			index:     i,
			filePath:  filepath.Join(d.tmpDir, fmt.Sprintf("%d_%s", i, d.name)),
			startByte: startByte,
			endByte:   endByte,
		}

		processList = append(processList, process)
	}

	d.processList = &processList
	return nil
}

func (d *Downloader) download() error {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout*time.Second)
	defer cancel()
	eg, egctx := errgroup.WithContext(ctx)

	for _, process := range *d.processList {
		p := process
		eg.Go(func() error {
			return p.download(egctx, d.url)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) merge() (err error) {
	dstFile, err := os.Create(d.name)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil {
			err = cerr
		}
	}()

	for _, process := range *d.processList {
		p := process
		if err := p.copy(dstFile); err != nil {
			return err
		}
	}

	return nil
}

func (p *process) download(ctx context.Context, url string) (err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", p.startByte, p.endByte))

	client := new(http.Client)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			err = cerr
		}
	}()

	tmpFile, err := os.Create(p.filePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := tmpFile.Close(); cerr != nil {
			err = cerr
		}
	}()

	if _, err := io.Copy(tmpFile, res.Body); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (p *process) copy(dstFile io.Writer) (err error) {
	tmpFile, err := os.Open(p.filePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := tmpFile.Close(); cerr != nil {
			err = cerr
		}
	}()

	if _, err := io.Copy(dstFile, tmpFile); err != nil {
		return err
	}
	return nil
}
