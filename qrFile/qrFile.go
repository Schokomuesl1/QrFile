package qrFile

import (
	"bufio"
	"bytes"
	"code.google.com/p/rsc/qr"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

const qrLevel = qr.L
const qrSize uint64 = 1608     // this needs to be even, since we encode binary using 2 hex chars
const qrHeaderSize uint64 = 60 // 3 uint64 as string
const qrDataSize uint64 = qrSize - qrHeaderSize
const uintStringLength = 20
const indexPos = 0
const maxIndexPos = 20
const payloadLengthPos = 40
const payloadPos = 60
const outputFormat = "%20d%20d%20d%s"
const payloadFormat = "%1548s"

type QrFile struct {
	Fname string
	Data  []byte
}

type QrElement struct {
	Index         uint64
	MaxIndex      uint64
	PayloadLength uint64 // nescessary to store this since we will pad up to max length
	Payload       string
}

type QrElements struct {
	Elements []QrElement
}

func (elem *QrElements) WritePNGs(workPath string, fnamePrefix string) error {
	control := make(chan error, len(elem.Elements))
	for i, v := range elem.Elements {
		v := v // we need to shadow v here so we work on copies
		go func(i int, v *QrElement) {
			log.Printf("Creating png for: %d %d %d %d |%s...|", i, v.Index, v.MaxIndex, v.PayloadLength, v.Payload[0:10])
			qr, err := v.AsQR()
			if err != nil {
				control <- err
			}

			imgByte := qr.PNG()
			img, _, _ := image.Decode(bytes.NewReader(imgByte))
			var fname = fmt.Sprintf("%s%s%d.png", workPath, fnamePrefix, i)
			out, err := os.Create(fname)
			/*if err != nil {
			    control <- err
			}*/
			err = png.Encode(out, img)
			if err != nil {
				control <- err
			}
			control <- nil
		}(i, &v)
	}
	errorList := make([]string, 0)
	for i := 0; i < len(elem.Elements); i++ {
		result := <-control
		if result != nil {
			errorList = append(errorList, result.Error())
		}
	}
	if len(errorList) == 0 {
		return nil
	}
	// concatenate the error messages & return
	return errors.New(strings.Join(errorList, "; "))
}

func (elem *QrElements) FromPNGs(workPath string, fnamePrefix string) error {
	dirContent, err := ioutil.ReadDir(workPath)
	if err != nil {
		return err
	}
	// spread this into goroutines, collect results afterwards
	control := make(chan *QrElement, len(dirContent))
	for _, v := range dirContent {
		go func(fname string) {
			if strings.Index(fname, fnamePrefix) == 0 && strings.Index(fname, ".png") == len(fname)-4 {
				newElement := new(QrElement)
				err := newElement.ParsePNG(fmt.Sprintf("%s/%s", workPath, fname))
				log.Print("Handling file ", fname)
				if err == nil {
					log.Printf("Element created: %d %d %d |%s...|", newElement.Index, newElement.MaxIndex, newElement.PayloadLength, newElement.Payload[0:10])
					control <- newElement
				} else {
					log.Print("No element created.")
					control <- nil
				}
			} else {
				log.Print("Not handling file ", fname)
				control <- nil // we have to notify also if we do not handle the file
			}
		}(v.Name())
	}

	// wait for all goroutines to return before starting
	for i := 0; i < len(dirContent); i++ {
		// consume the results
		result := <-control
		if result != nil {
			elem.Elements = append(elem.Elements, *result)
		} // else {
		// an error occurred, no appeding
		//}
	}

	log.Printf("Extracted %d elements", elem.Len())
	if len(elem.Elements) == 0 {
		return errors.New("No elements extraced.")
	}
	if uint64(elem.Len()) < elem.Elements[0].MaxIndex {
		return errors.New("Incomplete set extracted.")
	}
	sort.Sort(elem)
	// check that we have no duplicates
	for i := 0; i < elem.Len()-1; i++ {
		if !elem.Less(i, i+1) {
			return errors.New("Duplicate element detected.")
		}
	}
	return nil
}

func (elem *QrElements) StoreData(fileObject *QrFile) error {
	for _, v := range elem.Elements {
		log.Printf("Storing data for %d %d %d |%s...|", v.Index, v.MaxIndex, v.PayloadLength, v.Payload[0:10])
		buffer, err := hex.DecodeString(v.Payload)
		if err != nil {
			return err
		}
		fileObject.Data = append(fileObject.Data, buffer...)
	}
	return nil
}

func (elem *QrElement) ParsePNG(fname string) error {
	var result bytes.Buffer
	cmd := exec.Command("zbarimg", "--quiet", "-Sdisable", "-Sqrcode.enable", fname)
	cmd.Stdout = &result
	err := cmd.Run()
	if err != nil {
		return err
	}
	if elem.ParseString(strings.TrimSuffix(strings.TrimPrefix(result.String(), "QR-Code:"), "\n")) != nil {
		return err
	}
	return nil
}

// for sorting QrElements
func (elements *QrElements) Len() int { return len(elements.Elements) }
func (elements *QrElements) Swap(i, j int) {
	elements.Elements[i], elements.Elements[j] = elements.Elements[j], elements.Elements[i]
}
func (elements *QrElements) Less(i, j int) bool {
	return elements.Elements[i].Index < elements.Elements[j].Index
}

///
func MakeQrElements(elementCount uint64) *QrElements {
	qrf := new(QrElements)
	qrf.Elements = make([]QrElement, elementCount)
	return qrf
}

func GetElements(payload string) (elements *QrElements, err error) {
	var maxCount uint64 = uint64(len(payload)) / qrDataSize
	if uint64(len(payload))%qrDataSize != 0 {
		maxCount++
	}
	elements = MakeQrElements(maxCount)
	var i uint64
	for i = 0; i < maxCount; i++ {
		log.Printf("Creating element: %d %d", i, maxCount)
		if int((i+1)*qrDataSize) > len(payload) {
			elements.Elements[i], err = GetElement(i, maxCount-1, payload[i*qrDataSize:])
		} else {
			elements.Elements[i], err = GetElement(i, maxCount-1, payload[i*qrDataSize:(i+1)*qrDataSize])
		}
		if err != nil {
			return
		}
	}
	return
}

func GetElement(idx uint64, maxidx uint64, payload string) (elem QrElement, err error) {
	elem.Index = idx
	elem.MaxIndex = maxidx
	if len(payload) > int(qrDataSize) {
		return elem, errors.New("Payload size exceeds maximum data size")
	}
	elem.PayloadLength = uint64(len(payload))
	elem.Payload = fmt.Sprintf(payloadFormat, payload)
	return
}

func (elem *QrElement) AsString() string {
	return fmt.Sprintf(outputFormat, elem.Index, elem.MaxIndex, elem.PayloadLength, elem.Payload)
}

func (elem *QrElement) ParseString(str string) (err error) {
	if uint64(len(str)) != qrSize {
		return errors.New(fmt.Sprintf("Size mismatch. Expected %d, got %d!", qrSize, len(str)))
	}
	elem.Index, err = strconv.ParseUint(strings.Trim(str[indexPos:indexPos+uintStringLength], " "), 10, 16)
	if err != nil {
		return err
	}
	elem.MaxIndex, err = strconv.ParseUint(strings.Trim(str[maxIndexPos:maxIndexPos+uintStringLength], " "), 10, 16)
	if err != nil {
		return err
	}
	elem.PayloadLength, err = strconv.ParseUint(strings.Trim(str[payloadLengthPos:payloadLengthPos+uintStringLength], " "), 10, 16)
	if err != nil {
		return err
	}
	elem.Payload = string(strings.Trim(str[payloadPos:], " "))
	return nil
}

func (elem *QrElement) AsQR() (*qr.Code, error) {
	return qr.Encode(elem.AsString(), qrLevel)
}

func New() *QrFile {
	qrf := new(QrFile)
	qrf.Data = make([]byte, 0)
	return qrf
}

func FromFile(fname string) (*QrFile, error) {
	qrf := new(QrFile)
	qrf.Fname = fname
	err := qrf.ReadFile()
	return qrf, err
}

func (qrf *QrFile) ToHexString() (str string) {
	str = hex.EncodeToString(qrf.Data)
	return
}

func (qrf *QrFile) ToFile() (err error) {
	file, err := os.Create(qrf.Fname)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(qrf.Data)
	if err != nil {
		return err
	}
	return nil
}

func (qrf *QrFile) ReadFile() (err error) {
	file, err := os.Open(qrf.Fname)
	if err != nil {
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return
	}
	var size int64 = info.Size()
	qrf.Data = make([]byte, size)
	buffer := bufio.NewReader(file)
	_, err = buffer.Read(qrf.Data)
	return
}
