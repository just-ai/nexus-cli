package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"regexp"
	"strings"
	"strconv"

	"nexus-cli/registry"
	"github.com/urfave/cli"
	"github.com/blang/semver"
	"github.com/dustin/go-humanize"
)

const (
	credentialsTemplates = `# Nexus Credentials
nexus_host = "{{ .Host }}"
nexus_username = "{{ .Username }}"
nexus_password = "{{ .Password }}"
nexus_repository = "{{ .Repository }}"`
)

func main() {
	app := cli.NewApp()
	app.Name = "Nexus CLI"
	app.Usage = "Manage Docker Private Registry on Nexus"
	app.Version = "1.0.0-beta-2"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Mohamed Labouardy",
			Email: "mohamed@labouardy.com",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "configure",
			Usage: "Configure Nexus Credentials",
			Action: func(c *cli.Context) error {
				return setNexusCredentials(c)
			},
		},
		{
			Name:  "image",
			Usage: "Manage Docker Images",
			Subcommands: []cli.Command{
				{
					Name:  "ls",
					Usage: "List all images in repository",
					Flags: []cli.Flag{
						cli.StringSliceFlag{
							Name:  "expression, e",
							Usage: "Filter images by regular expression",
						},
						cli.BoolFlag{
							Name:  "invert, v",
							Usage: "Invert filter results",
						},
						cli.BoolFlag{
							Name:  "images-only, i",
							Usage: "Print only images, useful for scripts",
						},
					},
					Action: func(c *cli.Context) error {
						return listImages(c)
					},
				},
				{
					Name:  "tags",
					Usage: "Display all image tags",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "name, n",
							Usage: "List tags by image name",
						},
						cli.StringSliceFlag{
							Name:  "expression, e",
							Usage: "Filter tags by regular expression",
						},
						cli.BoolFlag{
							Name:  "invert, v",
							Usage: "Invert filter results",
						},
						cli.StringFlag{
							Name: "sort, s",
							Usage: "Sort tags by semantic version, assuming all tags are semver except latest.",
						},
					},
					Action: func(c *cli.Context) error {
						return listTagsByImage(c)
					},
				},
				{
					Name:  "info",
					Usage: "Show image details",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name: "name, n",
						},
						cli.StringFlag{
							Name: "tag, t",
						},
						cli.StringSliceFlag{
							Name:  "expression, e",
							Usage: "Filter tags by regular expression",
						},
						cli.BoolFlag{
							Name:  "invert, v",
							Usage: "Invert results filter expressions",
						},
						cli.BoolFlag{
							Name:	 "humanize",
							Usage: "Prints size as human readable",
						},
					},
					Action: func(c *cli.Context) error {
						return showImageInfo(c)
					},
				},
				{
					Name:  "delete",
					Usage: "Delete images",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name: "name, n",
						},
						cli.StringFlag{
							Name: "tag, t",
						},
						cli.StringFlag{
							Name: "keep, k",
						},
						cli.StringSliceFlag{
							Name:  "expression, e",
							Usage: "Filter tags by regular expression",
						},
						cli.BoolFlag{
							Name:  "invert, v",
							Usage: "Invert results filter expressions",
						},
						cli.StringFlag{
							Name: "sort, s",
						},
					},
					Action: func(c *cli.Context) error {
						return deleteImages(c)
					},
				},
				{
					Name:  "size",
					Usage: "Show total size of image including all tags",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name: "name, n",
						},
					},
					Action: func(c *cli.Context) error {
						return showTotalImageSize(c)
					},
				},
			},
		},
	}
	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "Wrong command %q !", command)
	}
	app.Run(os.Args)
}

func setNexusCredentials(c *cli.Context) error {
	var hostname, repository, username, password string
	fmt.Print("Enter Nexus Host: ")
	fmt.Scan(&hostname)
	fmt.Print("Enter Nexus Repository Name: ")
	fmt.Scan(&repository)
	fmt.Print("Enter Nexus Username: ")
	fmt.Scan(&username)
	fmt.Print("Enter Nexus Password: ")
	fmt.Scan(&password)

	data := struct {
		Host       string
		Username   string
		Password   string
		Repository string
	}{
		hostname,
		username,
		password,
		repository,
	}

	tmpl, err := template.New(".credentials").Parse(credentialsTemplates)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	f, err := os.Create(".credentials")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = tmpl.Execute(f, data)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

func listImages(c *cli.Context) error {
	r, err := registry.NewRegistry()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	images, err := r.ListImages()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	images, err = filterStringsByRegex(images, c.StringSlice("expression"), c.Bool("invert"))
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	for _, image := range images {
		fmt.Println(image)
	}
	if (!c.Bool("images-only")){
		fmt.Printf("Total images: %d\n", len(images))
	}
	return nil
}

func filterStringsByRegex(tags []string, expressions []string, invert bool) ([]string, error) {
	var retTags []string
	if len(expressions) == 0 {
		return tags, nil
	}
	for _, tag := range tags {
		tagMiss := false
		for _, expression := range expressions {
			var expressionBool = !invert
			if strings.HasPrefix(expression, "!") {
				expressionBool = invert
				expression = strings.Trim(expression, "!")
			}
			retVal, err := regexp.MatchString(expression, tag)
			if err != nil {
				return retTags, err
			}
			if retVal != expressionBool {
				tagMiss = true
				break
			}
		}
		// tag must match all expression, so continue with next tag on match
		if !tagMiss {
			retTags = append(retTags, tag)
		}
	}
	return retTags, nil
}

func listTagsByImage(c *cli.Context) error {
	var imgName = c.String("name")
	var sort = c.String("sort")
	if sort != "semver" {
		sort = "default"
	}

	r, err := registry.NewRegistry()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	if imgName == "" {
		cli.ShowSubcommandHelp(c)
	}
	tags, err := r.ListTagsByImage(imgName)

	// filter tags by expressions
	tags, err = filterStringsByRegex(tags, c.StringSlice("expression"), c.Bool("invert"))
	if err != nil {
		log.Fatal(err)
	}

	compareStringNumber := getSortComparisonStrategy(sort)
	Compare(compareStringNumber).Sort(tags)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	for _, tag := range tags {
		fmt.Println(tag)
	}
	fmt.Printf("There are %d images for %s\n", len(tags), imgName)
	return nil
}

func showImageInfo(c *cli.Context) error {
	var imgName = c.String("name")
	var tag = c.String("tag")
	r, err := registry.NewRegistry()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	if imgName == "" {
		cli.ShowSubcommandHelp(c)
	}
	var configSize int64
	var layers = make(map[string]int64)
	var totalLayersSize int64

	var handleTag = func(tag string) error {
		manifest, err := r.ImageManifest(imgName, tag)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		configSize += manifest.Config.Size
		for _, layer := range manifest.Layers {
			_, ok := layers[layer.Digest]
			if !ok {
				layers[layer.Digest] = layer.Size
			}
		}
	return nil
	}

	if tag == "" {
		tags, err := r.ListTagsByImage(imgName)

		// filter tags by expressions
		tags, err = filterStringsByRegex(tags, c.StringSlice("expression"), c.Bool("invert"))
		if err != nil {
			log.Fatal(err)
		}
		for _,tag := range tags {
			handleTag(tag)
		}
	} else {
		handleTag(tag)
	}

	var humanize = func(b int64) string {
		if (c.Bool("humanize")){
			return humanize.Bytes(uint64(b))
		}
		return strconv.FormatInt(b, 10)
	}

	fmt.Printf("Image: %s:%s\n", imgName, tag)
	fmt.Printf("Size: %s\n", humanize(configSize))
	fmt.Println("Layers:")
	for digest, size := range layers {
		totalLayersSize += size
		fmt.Printf("\t%s\t%s\n", digest, humanize(size))
	}
	fmt.Printf("Total layers size: %s\n", humanize(totalLayersSize))
	fmt.Printf("Total size: %s\n", humanize(totalLayersSize + configSize))
	return nil
}

func deleteImages(c *cli.Context) error {
	var imgName = c.String("name")
	var tag = c.String("tag")
	var keep = c.Int("keep")
	var invert = c.Bool("invert")

	// Show help if no image name is present
	var sort = c.String("sort")
	if sort != "semver" {
		sort = "default"
	}

	if imgName == "" {
		fmt.Fprintf(c.App.Writer, "You should specify the image name\n")
		cli.ShowSubcommandHelp(c)
		return nil
	}

	r, err := registry.NewRegistry()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// if a specific tag is provided, ignore all other options
	if tag != "" {
		err = r.DeleteImageByTag(imgName, tag)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}

	// Get list of tags and filter them by all expressions provided
	tags, err := r.ListTagsByImage(imgName)
	tags, err = filterStringsByRegex(tags, c.StringSlice("expression"), invert)
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Could not filter tags by regular expressions: %s\n", err)
		return err
	}

	// if no keep is specified, all flags are unset. Show help and exit.
	if c.IsSet("keep") == false && len(c.StringSlice("expression")) == 0 {
		fmt.Fprintf(c.App.Writer, "You should either specify use tag / filter expressions, or specify how many images you want to keep\n")
		cli.ShowSubcommandHelp(c)
		return fmt.Errorf("You should either specify use tag / filter expressions, or specify how many images you want to keep")
	}

	if len(tags) == 0 && !c.IsSet("keep") {
		fmt.Fprintf(c.App.Writer, "No images selected for deletion\n")
		return fmt.Errorf("No images selected for deletion")
	}

	// Remove images by using keep flag
	compareStringNumber := getSortComparisonStrategy(sort)
	Compare(compareStringNumber).Sort(tags)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	if len(tags) >= keep {
		for _, tag := range tags[:len(tags)-keep] {
			fmt.Printf("%s:%s image will be deleted ...\n", imgName, tag)
			err = r.DeleteImageByTag(imgName, tag)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
		}
	} else {
		fmt.Printf("Only %d images are available\n", len(tags))
	}
	return nil
}

func showTotalImageSize(c *cli.Context) error {
	var imgName = c.String("name")
	var totalSize (int64) = 0

	if imgName == "" {
		cli.ShowSubcommandHelp(c)
	} else {
		r, err := registry.NewRegistry()
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		tags, err := r.ListTagsByImage(imgName)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		for _, tag := range tags {
			manifest, err := r.ImageManifest(imgName, tag)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			sizeInfo := make(map[string]int64)

			for _, layer := range manifest.Layers {
				sizeInfo[layer.Digest] = layer.Size
			}

			for _, size := range sizeInfo {
				totalSize += size
			}
		}
		fmt.Printf("%d %s\n", totalSize, imgName)
	}
	return nil
}
func getSortComparisonStrategy(sort string) func(str1, str2 string) bool{
	var compareStringNumber func(str1, str2 string) bool

	if sort == "default" {
		compareStringNumber = func(str1, str2 string) bool {
			return extractNumberFromString(str1) < extractNumberFromString(str2)
		}
	}

	if sort == "semver" {
		compareStringNumber = func(str1, str2 string) bool {
			if str1 == "latest" {
				return false
			}
			if str2 == "latest" {
				return true
			}
			version1, err1 := semver.Make(str1)
			if err1 != nil {
			    fmt.Printf("Error parsing version1: %q\n", err1)
			}
			version2, err2 := semver.Make(str2)
			if err2 != nil {
			    fmt.Printf("Error parsing version2: %q\n", err2)
			}
			return version1.LT(version2)
		}
	}

	return compareStringNumber
}
