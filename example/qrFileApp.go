package main

import (
    "flag"
    "fmt"
    "github.com/Schokomuesl1/qrFile"
    "log"
    "strings"
)

func createQRFilesFromFile(inFile string, imgDir string, imgPrefix string) error {
    log.Printf("Creating QR codes for file %s into folder %s using image prefix %s.", inFile, imgDir, imgPrefix)
    qrf, err := qrFile.FromFile(inFile)
    if err != nil {
        return err
    }
    qrf.ReadFile()
    elements, err := qrFile.GetElements(qrf.ToHexString())
    if err != nil {
        return err
    }
    log.Printf("Successfully converted file to %d QR codes", len(elements.Elements))
    err = elements.WritePNGs(imgDir, imgPrefix)
    if err != nil {
        return err
    }
    log.Printf("Successfully wrote %d png files in %s.", len(elements.Elements), imgDir)
    return nil
}

func restoreFileFromQRImages(fileList []string, outputFilename string) error {
    log.Printf("Extracting data from input %s, writing to file %s.", strings.Join(fileList, ","), outputFilename)
    var newElem = new(qrFile.QrElements)
    err := newElem.FromPNGs(fileList)

    if err != nil {
        return err
    }
    log.Printf("Successfully read input files, now storing data...")
    newFile := new(qrFile.QrFile)
    newFile.Fname = outputFilename
    err = newElem.StoreData(newFile)
    if err != nil {
        return err
    }
    log.Printf("...done. Writing result file.")
    err = newFile.ToFile()
    if err != nil {
        return err
    }
    log.Printf("Done! Successfully wrote %s", outputFilename)
    return nil
}

func main() {
    var outDir string
    var imageDir string
    var inFile string
    var imagePrefix string
    var outFile string
    flag.StringVar(&outDir, "outputDirectory", "./output_dir", "Directory where result files are stored.")
    flag.StringVar(&imageDir, "imageDirectory", "./img_dir", "Directory where resulting image files")
    flag.StringVar(&imagePrefix, "imagePrefix", "img_", "Prefix of the resulting images in input mode.")
    flag.StringVar(&inFile, "in", "", "File to be converted in input mode. Providing an input file selects input mode.")
    flag.StringVar(&outFile, "out", "result", "File to store the extracted data to.")

    flag.Parse()

    if len(inFile) > 0 {
        err := createQRFilesFromFile(inFile, imageDir, imagePrefix)
        if err != nil {
            log.Fatalf("Error while handling input file %s: %s", inFile, err)
        }
    } else {
        // default to output mode
        if len(flag.Args()) == 0 {
            log.Fatal("Output modes requires at least one input file.")
        }
        err := restoreFileFromQRImages(flag.Args(), fmt.Sprintf("%s/%s", outDir, outFile))
        if err != nil {
            log.Fatalf("Error while handling output files %s: %s", flag.Args(), err)
        }
    }
}
