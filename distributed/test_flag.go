package main
import (
	"flag"
	"fmt"
)
func main() {
	width := flag.Int("width", 5, "width")
	flag.Parse()
	fmt.Printf("Width: %d\n", *width)
}
