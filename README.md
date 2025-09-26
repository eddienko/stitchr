# Stitchr

**Stitchr** is a Go command-line tool to create mosaics from TIFF images. It supports:

- Snake/serpentine scan patterns (vertical default, optional horizontal)
- Optional downsampling of large images
- Overlapping tiles with simple summing or alpha blending
- Selecting images via directory, regex filter, or explicit list

---

## Features

- **Vertical or horizontal snake patterns** for arranging tiles
- **Downsampling** to reduce memory usage
- **Overlaps** can be summed for additive effect
- Supports **TIFF input** and outputs **PNG mosaics**
- Prints progress (image filenames as they are processed)

---

## Installation

1. Make sure you have [Go](https://golang.org/dl/) installed.
2. Clone this repository:

```bash
git clone https://github.com/yourusername/stitchr.git
cd stitchr
````

3. Download dependencies:

```bash
go get golang.org/x/image/tiff
go get github.com/nfnt/resize
```

4. Build the executable:

```bash
go build -o stitchr stitchr.go
```

---

## Usage

```
./stitchr [options]
```

### Options

| Flag               | Description                                                  | Default      |
| ------------------ | ------------------------------------------------------------ | ------------ |
| `--dir string`     | Directory containing images (required unless using `--list`) |              |
| `--list string`    | Optional file containing a list of images                    |              |
| `--regex string`   | Optional regex to filter filenames in directory              |              |
| `--rows int`       | Number of rows in mosaic                                     |              |
| `--cols int`       | Number of columns in mosaic                                  |              |
| `--overlapX int`   | Overlap in X (pixels)                                        | 0            |
| `--overlapY int`   | Overlap in Y (pixels)                                        | 0            |
| `--downsample int` | Downsample factor (integer ≥1)                               | 1            |
| `--snake string`   | Snake pattern: `vertical` (default) or `horizontal`          | vertical     |
| `--out string`     | Output PNG file                                              | `mosaic.png` |

---

## Examples

**Default vertical snake pattern:**

```bash
./stitchr --dir ./images --rows 3 --cols 4 --overlapX 50 --overlapY 50 --downsample 4
```

**Horizontal snake pattern:**

```bash
./stitchr --dir ./images --rows 3 --cols 4 --snake horizontal --overlapX 50 --overlapY 50
```

**Using a list file for exact order:**

```bash
./stitchr --list images.txt --rows 2 --cols 2 --overlapX 20 --overlapY 20
```

**Filtering images with regex:**

```bash
./stitchr --dir ./images --regex "slice_.*\\.tif$" --rows 2 --cols 3
```

---

## Notes

* Only TIFF images are supported for input. Output is always PNG.
* The program prints each image filename as it is processed.
* Overlapping pixels can either be **added** or blended with alpha. Modify `blendImages` in the code to choose behavior.

---

## License

MIT License

---

## Author

Eduardo Gonzalez

