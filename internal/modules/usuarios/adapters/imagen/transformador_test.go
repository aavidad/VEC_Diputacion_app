package imagen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

func TestPNGRecortaCentroYEliminaMetadata(t *testing.T) {
	original := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{0, 190, 0, 255}
			if x < 2 {
				c = color.RGBA{220, 0, 0, 255}
			}
			if x >= 6 {
				c = color.RGBA{0, 0, 220, 255}
			}
			original.Set(x, y, c)
		}
	}
	var entrada bytes.Buffer
	if err := png.Encode(&entrada, original); err != nil {
		t.Fatal(err)
	}
	procesada, err := Nuevo().Procesar(context.Background(), entrada.Bytes(), ports.LimitesTransformacionImagen{})
	if err != nil {
		t.Fatal(err)
	}
	if procesada.TipoReal != "image/png" || procesada.TipoSalida != "image/png" || procesada.AnchoOriginal != 8 || procesada.AltoOriginal != 4 || procesada.Ancho != 256 || procesada.Alto != 256 || !procesada.OrientacionAplicada || !procesada.MetadatosEliminados {
		t.Fatalf("contrato de salida incorrecto: %+v", metadatos(procesada))
	}
	salida, err := png.Decode(bytes.NewReader(procesada.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []image.Point{{20, 128}, {128, 128}, {235, 128}} {
		r, g, b, _ := salida.At(p.X, p.Y).RGBA()
		if g < 30000 || r > 5000 || b > 5000 {
			t.Fatalf("el recorte no está centrado en %v: %d,%d,%d", p, r, g, b)
		}
	}
}

func TestJPEGOrientacionEXIFYLimpieza(t *testing.T) {
	original := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{230, 10, 10, 255}
			if x >= 20 {
				c = color.RGBA{10, 10, 230, 255}
			}
			original.Set(x, y, c)
		}
	}
	var jpegOriginal bytes.Buffer
	if err := jpeg.Encode(&jpegOriginal, original, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	conEXIF := anadirSegmentosJPEG(jpegOriginal.Bytes(), exifOrientacion6(), []byte("GPSLatitude=37.0; GPSLongitude=-3.0"))
	resultado, err := Nuevo().Procesar(context.Background(), conEXIF, ports.LimitesTransformacionImagen{})
	if err != nil {
		t.Fatal(err)
	}
	if resultado.TipoReal != "image/jpeg" || !resultado.OrientacionAplicada || !resultado.MetadatosEliminados {
		t.Fatalf("orientacion o tipo incorrecto: %+v", metadatos(resultado))
	}
	if bytes.Contains(resultado.Bytes, []byte("Exif")) || bytes.Contains(resultado.Bytes, []byte("GPSLatitude")) {
		t.Fatal("metadatos originales presentes en salida")
	}
	salida, err := png.Decode(bytes.NewReader(resultado.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	arribaR, _, arribaB, _ := salida.At(128, 20).RGBA()
	abajoR, _, abajoB, _ := salida.At(128, 235).RGBA()
	if arribaR < 40000 || arribaB > 12000 || abajoB < 40000 || abajoR > 12000 {
		t.Fatalf("orientacion 6 no aplicada antes del recorte: arriba %d/%d, abajo %d/%d", arribaR, arribaB, abajoR, abajoB)
	}
	conEXIF[12] = 0xff // Corrompe el orden de bytes TIFF; no se recupera la imagen a ciegas.
	if _, err := Nuevo().Procesar(context.Background(), conEXIF, ports.LimitesTransformacionImagen{}); !errors.Is(err, ErrImagenInvalida) {
		t.Fatalf("EXIF malformado aceptado: %v", err)
	}
}

func TestRechazaLimitesYTipoFalsoAntesDeDecodificar(t *testing.T) {
	ctx := context.Background()
	casos := []struct {
		name     string
		original []byte
	}{
		{"tamano", make([]byte, ports.TamanoMaximoOriginalImagen+1)},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{"gif", []byte("GIF89a")},
		{"mime_falso", append([]byte{0xff, 0xd8, 0xff}, []byte("not a jpeg")...)},
		{"dimension", pngConDimensiones(ports.DimensionMaximaOriginalImagen+1, 1)},
		{"pixeles", pngConDimensiones(5000, 5000)},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Nuevo().Procesar(ctx, tc.original, ports.LimitesTransformacionImagen{MaxBytes: 1 << 30, MaxDimension: 1 << 30, MaxPixeles: 1 << 30, LadoSalida: 1024}); !errors.Is(err, ErrImagenInvalida) {
				t.Fatalf("entrada adversarial aceptada: %v", err)
			}
		})
	}
}

func TestWebPSintetico(t *testing.T) {
	// WebP sin pérdidas 2x2 uniforme, generado a partir de píxeles sintéticos.
	b, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAdQhSLXo/+BiOh/AAA=")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Nuevo().Procesar(context.Background(), b, ports.LimitesTransformacionImagen{})
	if err != nil {
		t.Fatal(err)
	}
	if r.TipoReal != "image/webp" || r.TipoSalida != "image/png" || r.Ancho != 256 || r.Alto != 256 {
		t.Fatalf("WebP no transformado: %+v", metadatos(r))
	}
	if _, err := png.Decode(bytes.NewReader(r.Bytes)); err != nil {
		t.Fatal(err)
	}
}

func anadirSegmentosJPEG(jpg, exif, comentario []byte) []byte {
	var b bytes.Buffer
	b.Write(jpg[:2])
	segmento := func(marcador byte, contenido []byte) {
		b.Write([]byte{0xff, marcador})
		var longitud [2]byte
		binary.BigEndian.PutUint16(longitud[:], uint16(len(contenido)+2))
		b.Write(longitud[:])
		b.Write(contenido)
	}
	segmento(0xe1, exif)
	segmento(0xfe, comentario)
	b.Write(jpg[2:])
	return b.Bytes()
}

func exifOrientacion6() []byte {
	// IFD0: una etiqueta de orientación SHORT=6 y puntero IFD siguiente=0.
	b := make([]byte, 6+8+2+12+4)
	copy(b, "Exif\x00\x00II")
	binary.LittleEndian.PutUint16(b[8:10], 42)
	binary.LittleEndian.PutUint32(b[10:14], 8)
	binary.LittleEndian.PutUint16(b[14:16], 1)
	binary.LittleEndian.PutUint16(b[16:18], 0x0112)
	binary.LittleEndian.PutUint16(b[18:20], 3)
	binary.LittleEndian.PutUint32(b[20:24], 1)
	binary.LittleEndian.PutUint16(b[24:26], 6)
	return b
}

func pngConDimensiones(ancho, alto int) []byte {
	b := make([]byte, 8+4+4+13+4)
	copy(b, []byte("\x89PNG\r\n\x1a\n"))
	binary.BigEndian.PutUint32(b[8:12], 13)
	copy(b[12:16], "IHDR")
	binary.BigEndian.PutUint32(b[16:20], uint32(ancho))
	binary.BigEndian.PutUint32(b[20:24], uint32(alto))
	b[24] = 8
	b[25] = 6
	binary.BigEndian.PutUint32(b[29:33], crc32.ChecksumIEEE(b[12:29]))
	return b
}

func metadatos(i ports.ImagenProcesada) ports.ImagenProcesada { i.Bytes = nil; return i }
