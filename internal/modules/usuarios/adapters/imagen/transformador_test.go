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

// Fixtures WebP sintéticos (2x2 sin pérdidas, 2x2 con pérdidas y 40x20).
const (
	webpSinPerdidas = "UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAdQhSLXo/+BiOh/AAA="
	webpConPerdidas = "UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoCAAIAAMASJaACdLoB+AADsAD++3sX/zBK/IN5C//v3X99VPvqp/350AA="
	webp40x20       = "UklGRioAAABXRUJQVlA4TB4AAAAvJ8AEAA8wClfgPon5/EcLBAKEA/+1BgRE9D9K2AM="
)

func procesar(t *testing.T, b []byte) ports.FotoProcesada {
	t.Helper()
	r, err := Nuevo(1).Procesar(context.Background(), b)
	if err != nil {
		t.Fatalf("foto válida rechazada: %v", err)
	}
	salida, err := jpeg.Decode(bytes.NewReader(r.Bytes))
	if err != nil || salida.Bounds().Dx() != ports.LadoFotoImagen || salida.Bounds().Dy() != ports.LadoFotoImagen {
		t.Fatalf("salida no es un JPEG cuadrado de %d px: %v", ports.LadoFotoImagen, err)
	}
	if len(r.SHA256) != 64 || len(r.Bytes) > ports.TamanoMaximoFotoCustodia {
		t.Fatalf("huella o tamaño de salida incorrectos")
	}
	for _, marca := range [][]byte{[]byte("Exif"), []byte("eXIf"), []byte("EXIF"), []byte("GPS"), []byte("tEXt"), []byte("ICC_PROFILE")} {
		if bytes.Contains(r.Bytes, marca) {
			t.Fatalf("la salida conserva metadatos %q", marca)
		}
	}
	return r
}

func franjas(ancho, alto int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, ancho, alto))
	for y := 0; y < alto; y++ {
		for x := 0; x < ancho; x++ {
			c := color.RGBA{230, 10, 10, 255}
			if x >= ancho/2 {
				c = color.RGBA{10, 10, 230, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func TestPNGRecortaElCentro(t *testing.T) {
	original := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 800; x++ {
			c := color.RGBA{0, 190, 0, 255}
			if x < 200 {
				c = color.RGBA{220, 0, 0, 255}
			}
			if x >= 600 {
				c = color.RGBA{0, 0, 220, 255}
			}
			original.Set(x, y, c)
		}
	}
	var entrada bytes.Buffer
	if err := png.Encode(&entrada, original); err != nil {
		t.Fatal(err)
	}
	r := procesar(t, entrada.Bytes())
	salida, _ := jpeg.Decode(bytes.NewReader(r.Bytes))
	for _, p := range []image.Point{{20, 128}, {128, 128}, {235, 128}} {
		rr, g, b, _ := salida.At(p.X, p.Y).RGBA()
		if g < 30000 || rr > 12000 || b > 12000 {
			t.Fatalf("el recorte no está centrado en %v: %d,%d,%d", p, rr, g, b)
		}
	}
}

func TestJPEGAplicaOrientacionYQuitaEXIFyGPS(t *testing.T) {
	var base bytes.Buffer
	if err := jpeg.Encode(&base, franjas(40, 20), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	conEXIF := anadirSegmentosJPEG(base.Bytes(), exifOrientacion6(), []byte("GPSLatitude=37.0; GPSLongitude=-3.0"))
	r := procesar(t, conEXIF)
	salida, _ := jpeg.Decode(bytes.NewReader(r.Bytes))
	arribaR, _, arribaB, _ := salida.At(128, 20).RGBA()
	abajoR, _, abajoB, _ := salida.At(128, 235).RGBA()
	if arribaR < 40000 || arribaB > 12000 || abajoB < 40000 || abajoR > 12000 {
		t.Fatalf("orientación 6 no aplicada: arriba %d/%d, abajo %d/%d", arribaR, arribaB, abajoR, abajoB)
	}
	conEXIF[12] = 0xff // Orden TIFF imposible: no se decodifica a ciegas.
	if _, err := Nuevo(1).Procesar(context.Background(), conEXIF); !errors.Is(err, ports.ErrImagenFotoNoAdmitida) {
		t.Fatalf("EXIF malformado aceptado: %v", err)
	}
}

func TestOrientacionesDelCuadrado(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 3))
	src.Pix[src.PixOffset(0, 0)] = 255 // Marca roja en la esquina superior izquierda.
	esperada := map[uint8]image.Point{1: {0, 0}, 2: {2, 0}, 3: {2, 2}, 4: {0, 2}, 5: {0, 0}, 6: {2, 0}, 7: {2, 2}, 8: {0, 2}}
	for o, p := range esperada {
		dst := orientar(src, o)
		if dst.Pix[dst.PixOffset(p.X, p.Y)] != 255 {
			t.Fatalf("orientación %d no lleva la esquina a %v", o, p)
		}
	}
}

func TestWebPYPNGConEXIF(t *testing.T) {
	var pngBase bytes.Buffer
	if err := png.Encode(&pngBase, franjas(40, 20)); err != nil {
		t.Fatal(err)
	}
	webpBase, _ := base64.StdEncoding.DecodeString(webp40x20)
	for nombre, entrada := range map[string][]byte{
		"png":  pngConExif(pngBase.Bytes(), exifOrientacion6()[6:]),
		"webp": webpExtendido(40, 20, webpBase[12:], exifOrientacion6()[6:]),
	} {
		t.Run(nombre, func(t *testing.T) {
			r := procesar(t, entrada)
			salida, _ := jpeg.Decode(bytes.NewReader(r.Bytes))
			arribaR, _, arribaB, _ := salida.At(128, 20).RGBA()
			abajoR, _, abajoB, _ := salida.At(128, 235).RGBA()
			if arribaR < 40000 || arribaB > 12000 || abajoB < 40000 || abajoR > 12000 {
				t.Fatalf("EXIF no aplicado: arriba %d/%d, abajo %d/%d", arribaR, arribaB, abajoR, abajoB)
			}
		})
	}
	for _, b64 := range []string{webpSinPerdidas, webpConPerdidas} {
		b, _ := base64.StdEncoding.DecodeString(b64)
		procesar(t, b)
	}
}

func TestSalidaDeterminista(t *testing.T) {
	var base bytes.Buffer
	if err := png.Encode(&base, franjas(300, 200)); err != nil {
		t.Fatal(err)
	}
	a, b := procesar(t, base.Bytes()), procesar(t, base.Bytes())
	if a.SHA256 != b.SHA256 {
		t.Fatal("la misma foto produce salidas distintas: la repetición no se reconocería")
	}
}

func TestRechazaAntesDeDecodificar(t *testing.T) {
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, franjas(8, 8), nil); err != nil {
		t.Fatal(err)
	}
	muchasPasadas := append(append([]byte(nil), jpg.Bytes()...), bytes.Repeat([]byte{0xff, 0xda}, maxPasadasJPEG)...)
	webpBase, _ := base64.StdEncoding.DecodeString(webpSinPerdidas)
	oculto := append([]byte(nil), webpBase[12:]...)
	binary.LittleEndian.PutUint32(oculto[9:13], uint32(7999)|uint32(7999)<<14)
	webpOculto := webpExtendido(2, 2, oculto, nil)
	if c, err := webp.DecodeConfig(bytes.NewReader(webpOculto)); err != nil || c.Width != 2 {
		t.Fatal("el fixture debería ocultar el tamaño real tras VP8X")
	}
	casos := []struct {
		nombre   string
		entrada  []byte
		esperado error
	}{
		{"vacía", nil, ports.ErrImagenFotoNoAdmitida},
		{"fichero grande", make([]byte, ports.TamanoMaximoFotoImagen+1), ports.ErrImagenFotoGrande},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ports.ErrImagenFotoNoAdmitida},
		{"gif", []byte("GIF89a\x01\x00\x01\x00"), ports.ErrImagenFotoNoAdmitida},
		{"html con firma jpeg", append([]byte{0xff, 0xd8, 0xff}, []byte("<html>")...), ports.ErrImagenFotoNoAdmitida},
		{"ancho", pngCabecera(ports.DimensionMaximaFotoImagen+1, 1), ports.ErrImagenFotoGrande},
		{"píxeles", pngCabecera(5000, 5000), ports.ErrImagenFotoGrande},
		{"pasadas jpeg", muchasPasadas, ports.ErrImagenFotoNoAdmitida},
		{"webp con tamaño oculto", webpOculto, ports.ErrImagenFotoNoAdmitida},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if _, err := Nuevo(1).Procesar(context.Background(), c.entrada); !errors.Is(err, c.esperado) {
				t.Fatalf("esperado %v, obtenido %v", c.esperado, err)
			}
		})
	}
}

func TestEsperaTurnoYRespetaCancelacion(t *testing.T) {
	tr := Nuevo(1)
	tr.turnos <- struct{}{} // Otro proceso ocupa el único turno.
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	var b bytes.Buffer
	_ = png.Encode(&b, franjas(4, 4))
	if _, err := tr.Procesar(ctx, b.Bytes()); !errors.Is(err, context.Canceled) {
		t.Fatalf("sin turno debía esperar y respetar la cancelación: %v", err)
	}
}

// pngCabecera es un PNG bien formado (IHDR e IEND con CRC) que declara unas
// dimensiones sin aportar píxeles: sirve para probar el rechazo previo.
func pngCabecera(ancho, alto int) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	chunk := func(tipo string, datos []byte) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(datos)))
		b.Write(n[:])
		b.WriteString(tipo)
		b.Write(datos)
		binary.BigEndian.PutUint32(n[:], crc32.ChecksumIEEE(append([]byte(tipo), datos...)))
		b.Write(n[:])
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(ancho))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(alto))
	ihdr[8], ihdr[9] = 8, 6
	chunk("IHDR", ihdr)
	chunk("IEND", nil)
	return b.Bytes()
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
