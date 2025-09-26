package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/nfnt/resize"
	"golang.org/x/image/tiff"
)

func toGray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, img.At(x, y))
		}
	}
	return gray
}

// loadTIFF loads a TIFF image from disk
func loadTIFF(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := tiff.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// getImagePaths returns TIFF files from dir, optionally filtered by regex
func getImagePaths(dir string, regex *regexp.Regexp) ([]string, error) {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".tif" || filepath.Ext(path) == ".tiff") {
			if regex == nil || regex.MatchString(filepath.Base(path)) {
				paths = append(paths, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort paths for consistent ordering
	re := regexp.MustCompile(`-(\d+)_`)
	sort.Slice(paths, func(i, j int) bool {
		numI := 0
		numJ := 0

		mI := re.FindStringSubmatch(paths[i])
		if len(mI) > 1 {
			numI, _ = strconv.Atoi(mI[1])
		}

		mJ := re.FindStringSubmatch(paths[j])
		if len(mJ) > 1 {
			numJ, _ = strconv.Atoi(mJ[1])
		}

		return numI < numJ
	})

	return paths, nil
}

// loadListFile returns images listed in a text file (one per line)
func loadListFile(filename string) ([]string, error) {
	var paths []string
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			paths = append(paths, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

// sumImages adds src onto dst at position (x0,y0), summing RGB values
// sumImagesGray16 adds src onto dst at position (x0, y0), summing pixel values
func sumImages(dst *image.Gray16, src image.Image, x0, y0 int) {
	bounds := src.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			dstX := x0 + x
			dstY := y0 + y
			if dstX >= dst.Bounds().Dx() || dstY >= dst.Bounds().Dy() {
				continue
			}

			// Convert source pixel to 16-bit grayscale
			srcGray := color.Gray16Model.Convert(src.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray16)

			// Get current destination pixel
			dstGray := dst.Gray16At(dstX, dstY)

			// Sum and clamp to 65535
			sum := uint32(dstGray.Y) + uint32(srcGray.Y)
			if sum > 65535 {
				sum = 65535
			}

			dst.SetGray16(dstX, dstY, color.Gray16{uint16(sum)})
		}
	}
}

// mosaic creates the mosaic image in either vertical or horizontal snake pattern with blending
func mosaic(imgs []image.Image, rows, cols int, overlapX, overlapY int, snake string) (image.Image, error) {
	if len(imgs) != rows*cols {
		return nil, fmt.Errorf("number of images (%d) does not match grid size (%d)", len(imgs), rows*cols)
	}

	imgW := imgs[0].Bounds().Dx()
	imgH := imgs[0].Bounds().Dy()

	stepX := imgW - overlapX
	stepY := imgH - overlapY

	totalW := stepX*cols + overlapX
	totalH := stepY*rows + overlapY

	out := image.NewGray16(image.Rect(0, 0, totalW, totalH))

	idx := 0
	switch snake {
	case "horizontal":
		for r := 0; r < rows; r++ {
			if r%2 == 0 {
				// left → right
				for c := 0; c < cols; c++ {
					x := c * stepX
					y := r * stepY
					sumImages(out, imgs[idx], x, y)
					idx++
				}
			} else {
				// right → left
				for c := cols - 1; c >= 0; c-- {
					x := c * stepX
					y := r * stepY
					sumImages(out, imgs[idx], x, y)
					idx++
				}
			}
		}

	case "vertical", "":
		for c := 0; c < cols; c++ {
			if c%2 != 0 {
				// top → bottom
				for r := 0; r < rows; r++ {
					x := c * stepX
					y := r * stepY
					sumImages(out, imgs[idx], x, y)
					idx++
				}
			} else {
				// bottom → top
				for r := rows - 1; r >= 0; r-- {
					x := c * stepX
					y := r * stepY
					sumImages(out, imgs[idx], x, y)
					idx++
				}
			}
		}

	default:
		return nil, fmt.Errorf("invalid snake mode: %s (use 'vertical' or 'horizontal')", snake)
	}

	return out, nil
}

func main() {
	// Flags
	dir := flag.String("dir", "", "Directory containing images (required unless using --list)")
	rows := flag.Int("rows", 0, "Number of rows in mosaic")
	cols := flag.Int("cols", 0, "Number of columns in mosaic")
	overlapX := flag.Int("overlapX", 0, "Overlap in X (pixels)")
	overlapY := flag.Int("overlapY", 0, "Overlap in Y (pixels)")
	downsample := flag.Int("downsample", 1, "Downsample factor (integer >=1)")
	listFile := flag.String("list", "", "Optional file containing list of images")
	regexStr := flag.String("regex", "", "Optional regex to filter filenames in directory")
	output := flag.String("out", "mosaic.tiff", "Output TIFF file")
	snake := flag.String("snake", "vertical", "Snake pattern direction: vertical (default) or horizontal")

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// If no flags were provided, show usage and exit
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *rows <= 0 || *cols <= 0 {
		fmt.Println("Error: rows and cols must be > 0")
		flag.Usage()
		os.Exit(1)
	}
	if *downsample <= 0 {
		fmt.Println("downsample factor must be >= 1")
		flag.Usage()
		os.Exit(1)
	}

	var paths []string
	var err error

	if *listFile != "" {
		paths, err = loadListFile(*listFile)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if *dir == "" {
			fmt.Println("either --dir or --list must be specified")
			flag.Usage()
			os.Exit(1)
		}
		var regex *regexp.Regexp
		if *regexStr != "" {
			regex, err = regexp.Compile(*regexStr)
			if err != nil {
				fmt.Println("invalid regex:", err)
				flag.Usage()
				os.Exit(1)
			}
		}
		paths, err = getImagePaths(*dir, regex)
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(paths) < *rows**cols {
		log.Fatalf("Not enough images: have %d need %d", len(paths), *rows**cols)
	}

	var imgs []image.Image
	for _, p := range paths[:*rows**cols] {
		fmt.Printf("Processing %s\n", p)
		img, err := loadTIFF(p)
		if err != nil {
			log.Fatal(err)
		}
		if *downsample > 1 {
			w := uint(img.Bounds().Dx() / *downsample)
			h := uint(img.Bounds().Dy() / *downsample)
			img = resize.Resize(w, h, img, resize.Lanczos3)
		}
		imgs = append(imgs, img)
	}

	out, err := mosaic(imgs, *rows, *cols, *overlapX / *downsample, *overlapY / *downsample, *snake)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	opts := &tiff.Options{Compression: tiff.Deflate, Predictor: true} // optional compression
	if err := tiff.Encode(f, out, opts); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Mosaic saved as %s (grayscale TIFF)\n", *output)

	// f, err := os.Create(*output)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer f.Close()
	// if err := png.Encode(f, out); err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Printf("Mosaic saved as %s\n", *output)
}
