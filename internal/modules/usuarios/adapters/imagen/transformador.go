// Package imagen convierte la foto que sube una persona en el JPEG cuadrado
// que se muestra en la cabecera. Solo acepta JPEG, PNG y WebP reconocidos por
// su contenido (nunca por el nombre ni el tipo declarado), lee ancho y alto de
// la cabecera y rechaza antes de decodificar si superan los límites. La salida
// se recodifica desde los píxeles: no conserva EXIF, GPS, perfiles, textos ni
// ningún otro metadato del original.
package imagen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

// errFormato no sale del paquete: el caso de uso solo ve errores nominales.
var errFormato = errors.New("imagen: formato no admitido")

const (
	calidadJPEG = 88
	// Un JPEG progresivo normal tiene unas diez pasadas; miles de ellas
	// multiplican el coste de decodificar. Se cuentan antes de hacerlo.
	maxPasadasJPEG = 32
	// Escala intermedia: un primer paso rápido y otro de calidad.
	ladoIntermedio = 2 * ports.LadoFotoImagen
)

// Transformador limita cuántas fotos se decodifican a la vez: cada una puede
// ocupar decenas de MB durante unos cientos de milisegundos.
type Transformador struct {
	turnos chan struct{}
}

var _ ports.TransformadorFoto = (*Transformador)(nil)

// Nuevo admite como mucho `simultaneas` decodificaciones a la vez (1 a 8).
func Nuevo(simultaneas int) *Transformador {
	if simultaneas < 1 {
		simultaneas = 1
	}
	if simultaneas > 8 {
		simultaneas = 8
	}
	return &Transformador{turnos: make(chan struct{}, simultaneas)}
}

func (t *Transformador) Procesar(ctx context.Context, original []byte) (ports.FotoProcesada, error) {
	if t == nil || t.turnos == nil || ctx == nil {
		return ports.FotoProcesada{}, ports.ErrImagenNoDisponible
	}
	if len(original) > ports.TamanoMaximoFotoImagen {
		return ports.FotoProcesada{}, ports.ErrImagenFotoGrande
	}
	if len(original) == 0 {
		return ports.FotoProcesada{}, ports.ErrImagenFotoNoAdmitida
	}
	formato, orientacion, err := detectar(original)
	if err != nil {
		return ports.FotoProcesada{}, ports.ErrImagenFotoNoAdmitida
	}
	config, err := decodificarConfig(formato, original)
	if err != nil || config.Width < 1 || config.Height < 1 {
		return ports.FotoProcesada{}, ports.ErrImagenFotoNoAdmitida
	}
	if config.Width > ports.DimensionMaximaFotoImagen || config.Height > ports.DimensionMaximaFotoImagen ||
		int64(config.Width)*int64(config.Height) > ports.PixelesMaximosFotoImagen {
		return ports.FotoProcesada{}, ports.ErrImagenFotoGrande
	}
	select {
	case t.turnos <- struct{}{}:
		defer func() { <-t.turnos }()
	case <-ctx.Done():
		return ports.FotoProcesada{}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return ports.FotoProcesada{}, err
	}
	fuente, err := decodificar(formato, original)
	if err != nil || fuente.Bounds().Dx() != config.Width || fuente.Bounds().Dy() != config.Height {
		return ports.FotoProcesada{}, ports.ErrImagenFotoNoAdmitida
	}
	if err := ctx.Err(); err != nil {
		return ports.FotoProcesada{}, err
	}
	salida := orientar(cuadrado(fuente), orientacion)
	var b bytes.Buffer
	if err := jpeg.Encode(&b, salida, &jpeg.Options{Quality: calidadJPEG}); err != nil || b.Len() > ports.TamanoMaximoFotoCustodia {
		return ports.FotoProcesada{}, ports.ErrImagenNoDisponible
	}
	suma := sha256.Sum256(b.Bytes())
	return ports.FotoProcesada{Bytes: b.Bytes(), SHA256: hex.EncodeToString(suma[:])}, nil
}

// cuadrado recorta el centro y lo reduce a LadoFotoImagen sobre fondo blanco
// (la transparencia de PNG o WebP no existe en JPEG). Los dos pasos evitan
// muestrear millones de píxeles con el filtro caro.
func cuadrado(fuente image.Image) *image.RGBA {
	b := fuente.Bounds()
	lado := min(b.Dx(), b.Dy())
	x0, y0 := b.Min.X+(b.Dx()-lado)/2, b.Min.Y+(b.Dy()-lado)/2
	recorte := image.Rect(x0, y0, x0+lado, y0+lado)
	if lado > ladoIntermedio {
		intermedio := lienzoBlanco(ladoIntermedio)
		draw.ApproxBiLinear.Scale(intermedio, intermedio.Bounds(), fuente, recorte, draw.Over, nil)
		fuente, recorte = intermedio, intermedio.Bounds()
	}
	salida := lienzoBlanco(ports.LadoFotoImagen)
	draw.CatmullRom.Scale(salida, salida.Bounds(), fuente, recorte, draw.Over, nil)
	return salida
}

func lienzoBlanco(lado int) *image.RGBA {
	l := image.NewRGBA(image.Rect(0, 0, lado, lado))
	draw.Draw(l, l.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	return l
}

// orientar aplica la orientación EXIF (1–8) al cuadrado ya reducido. En un
// cuadrado centrado da lo mismo recortar antes o después de girar.
func orientar(src *image.RGBA, orientacion uint8) *image.RGBA {
	if orientacion <= 1 || orientacion > 8 {
		return src
	}
	n := src.Bounds().Dx()
	dst := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var ox, oy int
			switch orientacion {
			case 2:
				ox, oy = n-1-x, y
			case 3:
				ox, oy = n-1-x, n-1-y
			case 4:
				ox, oy = x, n-1-y
			case 5:
				ox, oy = y, x
			case 6:
				ox, oy = y, n-1-x
			case 7:
				ox, oy = n-1-y, n-1-x
			case 8:
				ox, oy = n-1-y, x
			}
			copy(dst.Pix[dst.PixOffset(x, y):dst.PixOffset(x, y)+4], src.Pix[src.PixOffset(ox, oy):src.PixOffset(ox, oy)+4])
		}
	}
	return dst
}

func detectar(b []byte) (formato string, orientacion uint8, err error) {
	switch {
	case len(b) >= 3 && bytes.Equal(b[:3], []byte{0xff, 0xd8, 0xff}):
		o, err := orientacionJPEG(b)
		if err == nil && bytes.Count(b, []byte{0xff, 0xda}) > maxPasadasJPEG {
			err = errFormato
		}
		return "jpeg", o, err
	case len(b) >= 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")):
		o, err := orientacionPNG(b)
		return "png", o, err
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		if uint64(binary.LittleEndian.Uint32(b[4:8]))+8 != uint64(len(b)) {
			return "", 0, errFormato
		}
		o, err := orientacionWebP(b)
		return "webp", o, err
	}
	return "", 0, errFormato
}

func decodificarConfig(formato string, b []byte) (image.Config, error) {
	r := bytes.NewReader(b)
	switch formato {
	case "jpeg":
		return jpeg.DecodeConfig(r)
	case "png":
		return png.DecodeConfig(r)
	case "webp":
		return webp.DecodeConfig(r)
	}
	return image.Config{}, errFormato
}

func decodificar(formato string, b []byte) (image.Image, error) {
	r := bytes.NewReader(b)
	switch formato {
	case "jpeg":
		return jpeg.Decode(r)
	case "png":
		return png.Decode(r)
	case "webp":
		return webp.Decode(r)
	}
	return nil, errFormato
}

// orientacionJPEG recorre los segmentos hasta el primer SOS. Solo acepta un
// EXIF TIFF bien delimitado; el resto de segmentos desaparece al recodificar.
func orientacionJPEG(b []byte) (uint8, error) {
	orientacion := uint8(1)
	encontrada := false
	for i := 2; i < len(b); {
		if b[i] != 0xff {
			return 0, errFormato
		}
		for i < len(b) && b[i] == 0xff {
			i++
		}
		if i >= len(b) {
			return 0, errFormato
		}
		marcador := b[i]
		i++
		if marcador == 0xda {
			return orientacion, nil
		}
		if marcador == 0xd9 {
			return 0, errFormato
		}
		if marcador == 0x01 || marcador >= 0xd0 && marcador <= 0xd7 {
			continue
		}
		if len(b)-i < 2 {
			return 0, errFormato
		}
		longitud := int(binary.BigEndian.Uint16(b[i : i+2]))
		if longitud < 2 || longitud > len(b)-i {
			return 0, errFormato
		}
		segmento := b[i+2 : i+longitud]
		if marcador == 0xe1 && (bytes.HasPrefix(segmento, []byte("Exif")) || bytes.HasPrefix(segmento, []byte("exif"))) {
			if encontrada || !bytes.HasPrefix(segmento, []byte("Exif\x00\x00")) {
				return 0, errFormato
			}
			var err error
			orientacion, err = leerOrientacionEXIF(segmento[6:])
			if err != nil {
				return 0, err
			}
			encontrada = true
		}
		i += longitud
	}
	return 0, errFormato
}

func leerOrientacionEXIF(tiff []byte) (uint8, error) {
	if len(tiff) < 8 {
		return 0, errFormato
	}
	var orden binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		orden = binary.LittleEndian
	case "MM":
		orden = binary.BigEndian
	default:
		return 0, errFormato
	}
	if orden.Uint16(tiff[2:4]) != 42 {
		return 0, errFormato
	}
	offset := uint64(orden.Uint32(tiff[4:8]))
	if offset > uint64(len(tiff)) || uint64(len(tiff))-offset < 2 {
		return 0, errFormato
	}
	numero := uint64(orden.Uint16(tiff[offset : offset+2]))
	if numero > (math.MaxUint64-offset-2)/12 || offset+2+numero*12+4 > uint64(len(tiff)) {
		return 0, errFormato
	}
	orientacion := uint8(1)
	vista := false
	for j := uint64(0); j < numero; j++ {
		entrada := tiff[offset+2+j*12 : offset+2+(j+1)*12]
		if orden.Uint16(entrada[:2]) != 0x0112 {
			continue
		}
		if vista || orden.Uint16(entrada[2:4]) != 3 || orden.Uint32(entrada[4:8]) != 1 {
			return 0, errFormato
		}
		v := orden.Uint16(entrada[8:10])
		if v < 1 || v > 8 {
			return 0, errFormato
		}
		orientacion = uint8(v)
		vista = true
	}
	return orientacion, nil
}
