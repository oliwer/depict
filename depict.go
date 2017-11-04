// DePict - Image Deduplication

package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/png"
	_ "image/jpeg"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

// Distance radius when searching similar images in the BKTree
const (
	Exact = 0
	Low = 4
	Medium = 8
	High = 16
	VeryHigh = 32
)

var accepted = []string{".jpg",".jpeg",".png",".gif"}

type ImageInfo struct {
	Hash BMVHash  `json:"h"`
	Name string   `json:"n"`
}

func (a ImageInfo) DistanceFrom(b ImageInfo) int {
	return a.Hash.HammingDistance(b.Hash)
}

func (ii ImageInfo) String() string {
	return fmt.Sprintf("%s  %s", ii.Name, ii.Hash)
}

func readImage(fn string) image.Image {
	file, err := os.Open(fn)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Decode the image.
	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	return img
}

func fingerprint(fn string) BMVHash {
	img := readImage(fn)
	return NewBMVHash(img)
}

func hasValidExt(fn string) bool {
	fn = strings.ToLower(fn)

	for i := range accepted {
		if strings.HasSuffix(fn, accepted[i]) {
			return true
		}
	}

	return false
}

func lookup(tree *BKTree, ii ImageInfo, radius int) {
	found := tree.Search(ii, radius)
	for i := range found {
		fmt.Printf("WARN %s is similar to %s\n", ii.Name, found[i].Name)
	}
}

func loadTree(path string) *BKTree {
	tree := new(BKTree)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return an empty tree
			return tree
		} else {
			panic(err)
		}
	}

	err = json.Unmarshal(bytes, tree)
	if err != nil {
		panic(err)
	}

	return tree
}

func saveTree(tree *BKTree, path string) {
	bytes, err := json.Marshal(tree)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		println("usage: depict <directory>")
		os.Exit(1)
	}
	dir := os.Args[1]
	dir = path.Clean(dir)

	dh, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	defer dh.Close()

	files, err := dh.Readdir(0)
	if err != nil {
		panic(err)
	}

	dbFile := path.Join(dir, "depict.db")
	tree := loadTree(dbFile)

	var wg sync.WaitGroup

	// Populate the database
	for i := range files {
		if files[i].IsDir() || !hasValidExt(files[i].Name()) {
			continue
		}
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			if tree.SearchByName(file) == nil {
				fp := fingerprint(path.Join(dir, file))
				tree.Add(ImageInfo{Hash: fp, Name: file})
			}
		}(files[i].Name())
	}
	wg.Wait()

	saveTree(tree, dbFile)

	// Look for similar images
	for k, v := range tree.SearchSimilars(Medium) {
		println(k, "is similar too:")
		for _, f := range v {
			println(" -", f)
		}
	}
}
