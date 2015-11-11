package main

import (
    "flag"
    "fmt"
    "github.com/Schokomuesl1/qrFile"
    "html/template"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
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

func httpHandler(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("template/index.html")
    t.Execute(w, nil)
}

func handleUploadedFile(w http.ResponseWriter, r *http.Request) {
    // the FormFile function takes in the POST input id file
    file, header, err := r.FormFile("file")
    log.Print("Handling request for uploaded file %s", file)

    if err != nil {
        log.Print(err)
        fmt.Fprintln(w, "An error occurred, please check log file.")
        return
    }

    defer file.Close()
    tempfile, err := ioutil.TempFile(os.TempDir(), "qrFileTemp")

    if err != nil {
        log.Print(w, "Unable to create the temporary file.")
        fmt.Fprintln(w, "An error occurred, please check log file.")
        return
    }
    log.Print("Created temporary file %s", tempfile.Name())

    // make sure to delete the file when we are done
    defer os.Remove(tempfile.Name())

    // copy POST data into the temporary file
    _, err = io.Copy(tempfile, file)
    if err != nil {
        log.Print(err)
        fmt.Fprintln(w, "An error occurred, please check log file.")
        return
    }

    // now process it, create qr images
    err = createQRFilesFromFile(tempfile.Name(), globTempDir, header.Filename+"_qr_")

    if err != nil {
        log.Print(w, "Error parsing files: %s", err.Error())
        fmt.Fprintln(w, "An error occurred, please check log file.")
        return
    }
    // store the image paths...
    images, _ := filepath.Glob(globTempDir + "/" + header.Filename + "_qr_*.png")
    for i, v := range images {
        images[i] = v[len(globTempDir+"/"):]
    }
    // fill page data struct with the relevant file names...
    pageData := struct {
        Filename string
        Images   []string
    }{Filename: header.Filename, Images: images}

    // and execute the template
    t, _ := template.ParseFiles("template/show.html")
    t.Execute(w, pageData)
}

var globTempDir string = ""

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
    interactive := flag.Bool("interactive", false, "If this is set, a small http server is started; the site provides a rudimentary interface to convert a file to QR images and display them.")
    port := flag.Int("port", 8080, "Http port for the web server.")

    flag.Parse()

    if *interactive {
        // start web server instance.
        log.Printf("Starting web server on port %d", *port)
        http.HandleFunc("/", httpHandler)
        http.HandleFunc("/receive/", handleUploadedFile)
        // create a temporary directory for the images:
        tempDir, err := ioutil.TempDir(os.TempDir(), "qrFileTempDir")
        if err != nil {
            log.Fatal("Unable to create temporary directory...")
            return
        }
        globTempDir = tempDir
        log.Print("TempDir: ", tempDir)
        // make sure we remove the temoporary directory as well...
        defer os.RemoveAll(tempDir)

        // serve the temporary folders contents as static data...
        http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(tempDir))))
        http.ListenAndServe(":"+strconv.Itoa(*port), nil)
    } else {
        if len(inFile) > 0 {
            err := createQRFilesFromFile(inFile, imageDir, imagePrefix)
            if err != nil {
                log.Fatalf("Error while handling input file %s: %s", inFile, err)
            }
        } else {
            // default to output mode
            if len(flag.Args()) == 0 {
                log.Fatal("Output mode requires at least one input file.")
            }
            err := restoreFileFromQRImages(flag.Args(), fmt.Sprintf("%s/%s", outDir, outFile))
            if err != nil {
                log.Fatalf("Error while handling output files %s: %s", flag.Args(), err)
            }
        }
    }
}
