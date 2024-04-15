package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"golang.org/x/net/html"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: furai <input file> <output file>")
		os.Exit(1)
	}

	if os.Args[1] == os.Args[2] {
		fmt.Fprintln(os.Stderr, "error: input file cannot be the same as output file")
		os.Exit(1)
	}

	in, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	tokenizer := html.NewTokenizer(in)
	_, err = compile(os.Args[1], "", tokenizer, "", "", nil, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

}

// Compiles some HTML. Stops when closingTagName is reached, if closingTagName is not empty.
// Reads body content from body if it is non-nil and needed. Writes output to out.
func compile(containingFilePath string, closingTagName string, tokenizer *html.Tokenizer, bodyContainingFilePath string, bodyClosingTagName string, bodyTokenizer *html.Tokenizer, out io.Writer) (consumedSlot bool, err error) {
loop:
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				break loop
			} else {
				err = tokenizer.Err()
				return
			}
		case html.StartTagToken:
			tagNameBytes, _ := tokenizer.TagName()
			componentTagName := string(tagNameBytes)
			componentPath := componentPath(containingFilePath, componentTagName)

			if componentTagName == "slot" {
				if bodyTokenizer == nil || bodyContainingFilePath == "" {
					// use default content inside slot element if we don't have a body
					continue loop
				}

				_, err = compile(bodyContainingFilePath, bodyClosingTagName, bodyTokenizer, "", "", nil, out)
				if err != nil {
					return
				}

				consumedSlot = true

				continue loop
			} else if _, statErr := os.Stat(componentPath); statErr == nil {
				var componentFile *os.File
				componentFile, err = os.Open(componentPath)
				if err != nil {
					return
				}
				componentTokenizer := html.NewTokenizer(componentFile)
				var componentConsumedSlot bool
				componentConsumedSlot, err = compile(componentPath, "", componentTokenizer, containingFilePath, componentTagName, tokenizer, out)
				if err != nil {
					return
				}

				if !componentConsumedSlot {
					tokenizer.Next()
					tagName, _ := tokenizer.TagName()
					if tokenizer.Token().Type != html.EndTagToken || string(tagName) != componentTagName {
						err = errors.New("attempted to provide content to slotless component")
						return
					}
				}

				continue loop
			}
		case html.EndTagToken:
			tagNameBytes, _ := tokenizer.TagName()
			tagName := string(tagNameBytes)
			if closingTagName != "" && tagName == closingTagName {
				// we're done
				return
			} else if tagName == "slot" {
				// skip over closing slot elements
				continue loop
			}
		}

		_, err = out.Write(tokenizer.Raw())
		if err != nil {
			return
		}
	}

	return
}

func componentPath(from, name string) string {
	return path.Join(path.Dir(from), name)
}
