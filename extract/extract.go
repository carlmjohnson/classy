package extract

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlmjohnson/flagx"
	"github.com/carlmjohnson/versioninfo"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html"
)

const AppName = "Classy"

func CLI(args []string) error {
	var app appEnv
	err := app.ParseArgs(args)
	if err != nil {
		return err
	}
	if err = app.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)
	fl.IntVar(&app.names, "names", 1, "")
	fl.IntVar(&app.threshold, "threshold", 3, "")
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `classy - %s

Find longest common class name combinations

Usage:

	classy [options] <source dir>

Options:
`, versioninfo.Version)
		fl.PrintDefaults()
	}
	if err := fl.Parse(args); err != nil {
		return err
	}
	if err := flagx.ParseEnv(fl, AppName); err != nil {
		return err
	}
	if err := flagx.MustHaveArgs(fl, 1, 1); err != nil {
		return err
	}
	app.srcDir = fl.Arg(0)

	return nil
}

type appEnv struct {
	srcDir    string
	names     int
	threshold int
}

func (app *appEnv) Exec() (err error) {
	files, err := app.getFiles()
	if err != nil {
		return err
	}
	var allClasses []string
	for _, file := range files {
		classes, err := app.classesInFile(file)
		if err != nil {
			return err
		}
		allClasses = append(allClasses, classes...)
	}
	counts := map[string]int{}
	for _, class := range allClasses {
		counts[class]++
	}
	slices.Sort(allClasses)
	allClasses = slices.Compact(allClasses)
	slices.SortFunc(allClasses, func(a, b string) bool {
		return counts[a] < counts[b]
	})
	for _, class := range allClasses {
		spaces := strings.Count(class, " ")
		if spaces < app.names {
			continue
		}
		count := counts[class]
		if count < app.threshold {
			continue
		}
		fmt.Printf("%2d\t%q\n", count, class)
	}
	return nil
}

func (app *appEnv) getFiles() (files []string, err error) {
	err = filepath.WalkDir(app.srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if ext := filepath.Ext(d.Name()); ext == ".html" || ext == ".htm" {
			files = append(files, path)
		}
		return nil
	})
	return
}

func (app *appEnv) classesInFile(name string) (classes []string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	n, err := html.Parse(f)
	if err != nil {
		return nil, err
	}

	return app.classSets(n)
}

func (app *appEnv) classSets(doc *html.Node) (sets []string, err error) {
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "class" {
					sets = append(sets, classSet(a.Val))
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return
}

func classSet(class string) string {
	for {
		prefix, interior, found := strings.Cut(class, "{{")
		if !found {
			break
		}
		var suffix string
		_, suffix, _ = strings.Cut(interior, "}}")
		class = prefix + " " + suffix
	}
	items := strings.Fields(class)
	slices.Sort(items)
	items = slices.Compact(items)
	return strings.Join(items, " ")
}
