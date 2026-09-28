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

	"golang.org/x/image/webp"

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
			if _, err := Nuevo().Procesar(ctx, tc.original, ports.LimitesTransformacionImagen{MaxBytes: 1 << 30, MaxDimension: 1 << 30, MaxPixeles: 1 << 30, LadoSalida: 1024}); !errors.Is(err, ports.ErrImagenPeticionInvalida) {
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

func TestWebPVP8ConPerdidas(t *testing.T) {
	b, err := base64.StdEncoding.DecodeString("UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoCAAIAAMASJaACdLoB+AADsAD++3sX/zBK/IN5C//v3X99VPvqp/350AA=")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Nuevo().Procesar(context.Background(), b, ports.LimitesTransformacionImagen{})
	if err != nil || r.TipoReal != "image/webp" {
		t.Fatalf("WebP VP8 válido rechazado: %v", err)
	}
}

func TestVP8XNoOcultaDimensionesVP8L(t *testing.T) {
	// DecodeConfig de WebP devuelve el canvas VP8X. La cabecera VP8L interna
	// declara 8192x8192 y debe rechazarse antes de llamar a Decode.
	base, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAdQhSLXo/+BiOh/AAA=")
	if err != nil {
		t.Fatal(err)
	}
	chunk := append([]byte(nil), base[12:]...)
	if string(chunk[:4]) != "VP8L" {
		t.Fatal("fixture VP8L incorrecto")
	}
	binary.LittleEndian.PutUint32(chunk[9:13], uint32(8191)|uint32(8191)<<14)
	adversarial := webpExtendido(2, 2, chunk, nil)
	config, err := webp.DecodeConfig(bytes.NewReader(adversarial))
	if err != nil || config.Width != 2 || config.Height != 2 {
		t.Fatalf("fixture no oculta tamaño: %+v %v", config, err)
	}
	if _, err := Nuevo().Procesar(context.Background(), adversarial, ports.LimitesTransformacionImagen{}); !errors.Is(err, ErrImagenInvalida) {
		t.Fatalf("VP8L con dimensión oculta aceptado: %v", err)
	}
}

func TestOrientacionEXIFEnPNGWebP(t *testing.T) {
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
	var pngBase bytes.Buffer
	if err := png.Encode(&pngBase, original); err != nil {
		t.Fatal(err)
	}
	pngExif := pngConExif(pngBase.Bytes(), exifOrientacion6()[6:])
	webpBase, err := base64.StdEncoding.DecodeString("UklGRioAAABXRUJQVlA4TB4AAAAvJ8AEAA8wClfgPon5/EcLBAKEA/+1BgRE9D9K2AM=")
	if err != nil {
		t.Fatal(err)
	}
	webpExif := webpExtendido(40, 20, webpBase[12:], exifOrientacion6()[6:])
	for _, tc := range []struct {
		name     string
		original []byte
		tipo     string
	}{{"png", pngExif, "image/png"}, {"webp", webpExif, "image/webp"}} {
		t.Run(tc.name, func(t *testing.T) {
			procesada, err := Nuevo().Procesar(context.Background(), tc.original, ports.LimitesTransformacionImagen{})
			if err != nil {
				t.Fatal(err)
			}
			if procesada.TipoReal != tc.tipo || bytes.Contains(procesada.Bytes, []byte("Exif")) || bytes.Contains(procesada.Bytes, []byte("eXIf")) || bytes.Contains(procesada.Bytes, []byte("EXIF")) {
				t.Fatalf("tipo o limpieza incorrecta: %+v", metadatos(procesada))
			}
			salida, err := png.Decode(bytes.NewReader(procesada.Bytes))
			if err != nil {
				t.Fatal(err)
			}
			arribaR, _, arribaB, _ := salida.At(128, 20).RGBA()
			abajoR, _, abajoB, _ := salida.At(128, 235).RGBA()
			if arribaR < 40000 || arribaB > 12000 || abajoB < 40000 || abajoR > 12000 {
				t.Fatalf("EXIF no aplicado antes del recorte: arriba %d/%d, abajo %d/%d", arribaR, arribaB, abajoR, abajoB)
			}
		})
	}
}

func TestRechazaEXIFMalformadoEnPNGWebP(t *testing.T) {
	var basePNG bytes.Buffer
	if err := png.Encode(&basePNG, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	baseWebP, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAdQhSLXo/+BiOh/AAA=")
	if err != nil {
		t.Fatal(err)
	}
	exif := append([]byte(nil), exifOrientacion6()[6:]...)
	exif[0] = 0xff // Orden TIFF imposible, con CRC y RIFF válidos.
	for _, entrada := range [][]byte{pngConExif(basePNG.Bytes(), exif), webpExtendido(2, 2, baseWebP[12:], exif)} {
		if _, err := Nuevo().Procesar(context.Background(), entrada, ports.LimitesTransformacionImagen{}); !errors.Is(err, ports.ErrImagenPeticionInvalida) {
			t.Fatalf("EXIF malformado aceptado o error incorrecto: %v", err)
		}
	}
}

func webpExtendido(ancho, alto int, imagenChunk, exif []byte) []byte {
	var chunks bytes.Buffer
	chunk := func(nombre string, contenido []byte) {
		chunks.WriteString(nombre)
		var n [4]byte
		binary.LittleEndian.PutUint32(n[:], uint32(len(contenido)))
		chunks.Write(n[:])
		chunks.Write(contenido)
		if len(contenido)%2 != 0 {
			chunks.WriteByte(0)
		}
	}
	encabezado := make([]byte, 10)
	if len(exif) > 0 {
		encabezado[0] = 0x08
	}
	a := ancho - 1
	h := alto - 1
	encabezado[4], encabezado[5], encabezado[6] = byte(a), byte(a>>8), byte(a>>16)
	encabezado[7], encabezado[8], encabezado[9] = byte(h), byte(h>>8), byte(h>>16)
	chunk("VP8X", encabezado)
	chunks.Write(imagenChunk)
	if len(exif) > 0 {
		chunk("EXIF", exif)
	}
	var salida bytes.Buffer
	salida.WriteString("RIFF")
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(chunks.Len()+4))
	salida.Write(n[:])
	salida.WriteString("WEBP")
	salida.Write(chunks.Bytes())
	return salida.Bytes()
}

func pngConExif(base, exif []byte) []byte {
	var b bytes.Buffer
	b.Write(base[:33]) // Firma e IHDR completo.
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(exif)))
	b.Write(n[:])
	b.WriteString("eXIf")
	b.Write(exif)
	binary.BigEndian.PutUint32(n[:], crc32.ChecksumIEEE(append([]byte("eXIf"), exif...)))
	b.Write(n[:])
	b.Write(base[33:])
	return b.Bytes()
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
