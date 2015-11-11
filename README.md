# qrFile

qrFile provides operations to convert a file to a set of QR code images and eventually restore this file from the image set. The functionality is contained in the qrFile package.

## Sample implementation

A small command line tool is included in the example folder.

    Command line args for qrFileApp
    -imageDirectory string
        Directory where resulting image files (default "./img_dir")
    -imagePrefix string
        Prefix of the resulting images in input mode. (default "img_")
    -in string
        File to be converted in input mode. Providing an input file selects input mode.
    -interactive
        If this is set, a small http server is started; the site provides a rudimentary interface to convert a file to QR images and display them.
    -out string
        File to store the extracted data to. (default "result")
    -outputDirectory string
        Directory where result files are stored. (default "./output_dir")
    -port int
        Http port for the web server. (default 8080)

If provided with the --in parameter, qrFileApp converts the file provided into a set of png images containing the file contents encoded in QR images.

    go run qrFileAPp.go --in ~/test.txt

If no named arguments are provided, qrFileApp reads the argument list as a file list containing images. It then tries to restore the contained data, writing the results into the default folder (./output_dir) using the default filename (result).

    go run qrFileAPp.go img_dir/*

The tool can be started with the --interactive flag (and, optionally, a --port flag). If so, a _very_ rudimentary web server is started which provides an interface to encode a file and display the resulting data.

    go run qrFileAPp.go --interactive