package imagen

import (
	"bytes"
	"context"
	"encoding/binary"
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

// ErrImagenInvalida no incorpora bytes, metadatos ni detalles del archivo original.
var ErrImagenInvalida = errors.New("imagen no admitida")

type Transformador struct{}

var _ ports.TransformadorImagen = Transformador{}

func Nuevo() Transformador { return Transformador{} }

func (Transformador) Procesar(ctx context.Context, original []byte, solicitados ports.LimitesTransformacionImagen) (ports.ImagenProcesada, error) {
	limites := restringir(solicitados)
	if ctx == nil || ctx.Err() != nil || len(original) == 0 || len(original) > limites.MaxBytes {
		return ports.ImagenProcesada{}, ErrImagenInvalida
	}
	formato, tipo, orientacion, err := detectar(original)
	if err != nil {
		return ports.ImagenProcesada{}, ErrImagenInvalida
	}
	config, err := decodificarConfig(formato, bytes.NewReader(original))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > limites.MaxDimension || config.Height > limites.MaxDimension || int64(config.Width)*int64(config.Height) > int64(limites.MaxPixeles) {
		return ports.ImagenProcesada{}, ErrImagenInvalida
	}
	if ctx.Err() != nil {
		return ports.ImagenProcesada{}, ctx.Err()
	}
	decodificada, err := decodificar(formato, bytes.NewReader(original))
	if err != nil || decodificada.Bounds().Dx() != config.Width || decodificada.Bounds().Dy() != config.Height {
		return ports.ImagenProcesada{}, ErrImagenInvalida
	}
	if ctx.Err() != nil {
		return ports.ImagenProcesada{}, ctx.Err()
	}
	orientada := vistaOrientada{Image: decodificada, orientacion: orientacion}
	ancho, alto := orientada.dimensiones()
	lado := min(ancho, alto)
	x0, y0 := (ancho-lado)/2, (alto-lado)/2
	salida := image.NewRGBA(image.Rect(0, 0, limites.LadoSalida, limites.LadoSalida))
	draw.CatmullRom.Scale(salida, salida.Bounds(), orientada, image.Rect(x0, y0, x0+lado, y0+lado), draw.Src, nil)
	if ctx.Err() != nil {
		return ports.ImagenProcesada{}, ctx.Err()
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, salida); err != nil {
		return ports.ImagenProcesada{}, ErrImagenInvalida
	}
	return ports.ImagenProcesada{
		Bytes: buffer.Bytes(), TipoReal: tipo, TipoSalida: "image/png",
		AnchoOriginal: config.Width, AltoOriginal: config.Height,
		Ancho: limites.LadoSalida, Alto: limites.LadoSalida,
		// La orientación identidad también queda resuelta: el consumidor exige
		// esta garantía incluso cuando la entrada carece de EXIF.
		OrientacionAplicada: true, MetadatosEliminados: true,
	}, nil
}

func restringir(v ports.LimitesTransformacionImagen) ports.LimitesTransformacionImagen {
	return ports.LimitesTransformacionImagen{
		MaxBytes:     acotar(v.MaxBytes, ports.TamanoMaximoOriginalImagen),
		MaxDimension: acotar(v.MaxDimension, ports.DimensionMaximaOriginalImagen),
		MaxPixeles:   acotar(v.MaxPixeles, ports.PixelesMaximosOriginalImagen),
		LadoSalida:   ports.LadoImagenProcesada,
	}
}

func acotar(v, techo int) int {
	if v <= 0 || v > techo {
		return techo
	}
	return v
}

func detectar(b []byte) (formato, tipo string, orientacion uint8, err error) {
	switch {
	case len(b) >= 3 && bytes.Equal(b[:3], []byte{0xff, 0xd8, 0xff}):
		o, err := orientacionJPEG(b)
		return "jpeg", "image/jpeg", o, err
	case len(b) >= 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "png", "image/png", 1, nil
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		if uint64(binary.LittleEndian.Uint32(b[4:8]))+8 != uint64(len(b)) {
			return "", "", 0, ErrImagenInvalida
		}
		return "webp", "image/webp", 1, nil
	default:
		return "", "", 0, ErrImagenInvalida
	}
}

func decodificarConfig(formato string, r *bytes.Reader) (image.Config, error) {
	switch formato {
	case "jpeg":
		return jpeg.DecodeConfig(r)
	case "png":
		return png.DecodeConfig(r)
	case "webp":
		return webp.DecodeConfig(r)
	default:
		return image.Config{}, ErrImagenInvalida
	}
}

func decodificar(formato string, r *bytes.Reader) (image.Image, error) {
	switch formato {
	case "jpeg":
		return jpeg.Decode(r)
	case "png":
		return png.Decode(r)
	case "webp":
		return webp.Decode(r)
	default:
		return nil, ErrImagenInvalida
	}
}

// Sólo se acepta EXIF TIFF bien delimitado y una orientación válida en IFD0.
// Otras secciones APP1 no alteran los píxeles y desaparecen al recodificar.
func orientacionJPEG(b []byte) (uint8, error) {
	orientacion := uint8(1)
	encontrada := false
	for i := 2; i < len(b); {
		if b[i] != 0xff {
			return 0, ErrImagenInvalida
		}
		for i < len(b) && b[i] == 0xff {
			i++
		}
		if i >= len(b) {
			return 0, ErrImagenInvalida
		}
		marcador := b[i]
		i++
		if marcador == 0xda {
			return orientacion, nil
		}
		if marcador == 0xd9 {
			return 0, ErrImagenInvalida
		}
		if marcador == 0x01 || marcador >= 0xd0 && marcador <= 0xd7 {
			continue
		}
		if len(b)-i < 2 {
			return 0, ErrImagenInvalida
		}
		longitud := int(binary.BigEndian.Uint16(b[i : i+2]))
		if longitud < 2 || longitud > len(b)-i {
			return 0, ErrImagenInvalida
		}
		segmento := b[i+2 : i+longitud]
		if marcador == 0xe1 && (bytes.HasPrefix(segmento, []byte("Exif")) || bytes.HasPrefix(segmento, []byte("exif"))) {
			if encontrada || !bytes.HasPrefix(segmento, []byte("Exif\x00\x00")) {
				return 0, ErrImagenInvalida
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
	return 0, ErrImagenInvalida
}

func leerOrientacionEXIF(tiff []byte) (uint8, error) {
	if len(tiff) < 8 {
		return 0, ErrImagenInvalida
	}
	var orden binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		orden = binary.LittleEndian
	case "MM":
		orden = binary.BigEndian
	default:
		return 0, ErrImagenInvalida
	}
	if orden.Uint16(tiff[2:4]) != 42 {
		return 0, ErrImagenInvalida
	}
	offset := uint64(orden.Uint32(tiff[4:8]))
	if offset > uint64(len(tiff)) || uint64(len(tiff))-offset < 2 {
		return 0, ErrImagenInvalida
	}
	numero := uint64(orden.Uint16(tiff[offset : offset+2]))
	if numero > (math.MaxUint64-offset-2)/12 || offset+2+numero*12+4 > uint64(len(tiff)) {
		return 0, ErrImagenInvalida
	}
	orientacion := uint8(1)
	inspeccionada := false
	for j := uint64(0); j < numero; j++ {
		entrada := tiff[offset+2+j*12 : offset+2+(j+1)*12]
		if orden.Uint16(entrada[:2]) != 0x0112 {
			continue
		}
		if inspeccionada || orden.Uint16(entrada[2:4]) != 3 || orden.Uint32(entrada[4:8]) != 1 {
			return 0, ErrImagenInvalida
		}
		v := orden.Uint16(entrada[8:10])
		if v < 1 || v > 8 {
			return 0, ErrImagenInvalida
		}
		orientacion = uint8(v)
		inspeccionada = true
	}
	return orientacion, nil
}

type vistaOrientada struct {
	image.Image
	orientacion uint8
}

func (v vistaOrientada) dimensiones() (int, int) {
	b := v.Image.Bounds()
	if v.orientacion >= 5 {
		return b.Dy(), b.Dx()
	}
	return b.Dx(), b.Dy()
}
func (v vistaOrientada) Bounds() image.Rectangle {
	w, h := v.dimensiones()
	return image.Rect(0, 0, w, h)
}
func (v vistaOrientada) At(x, y int) color.Color {
	b := v.Image.Bounds()
	w, h := b.Dx(), b.Dy()
	if !image.Pt(x, y).In(v.Bounds()) {
		return color.RGBA{}
	}
	var ox, oy int
	switch v.orientacion {
	case 2:
		ox, oy = w-1-x, y
	case 3:
		ox, oy = w-1-x, h-1-y
	case 4:
		ox, oy = x, h-1-y
	case 5:
		ox, oy = y, x
	case 6:
		ox, oy = y, h-1-x
	case 7:
		ox, oy = w-1-y, h-1-x
	case 8:
		ox, oy = w-1-y, x
	default:
		ox, oy = x, y
	}
	return v.Image.At(b.Min.X+ox, b.Min.Y+oy)
}
