package imagen

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"

	"golang.org/x/image/vp8"
	"golang.org/x/image/vp8l"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

// Se recorren todos los chunks antes de decodificar. DecodeConfig de WebP
// puede devolver sólo el canvas VP8X sin leer la cabecera de píxeles interna.
func orientacionWebP(b []byte) (uint8, error) {
	var anchoCanvas, altoCanvas int
	var extendido, imagenVista, exifVisto, exifEsperado bool
	orientacion := uint8(1)
	for i := 12; i < len(b); {
		if len(b)-i < 8 {
			return 0, errFormato
		}
		longitud := uint64(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		if longitud > uint64(len(b)-i-8) || longitud+longitud%2 > uint64(len(b)-i-8) {
			return 0, errFormato
		}
		chunk := b[i+8 : i+8+int(longitud)]
		switch string(b[i : i+4]) {
		case "VP8X":
			if i != 12 || extendido || len(chunk) != 10 || chunk[0]&0xc1 != 0 || chunk[1] != 0 || chunk[2] != 0 || chunk[3] != 0 || chunk[0]&0x02 != 0 {
				return 0, errFormato
			}
			extendido = true
			exifEsperado = chunk[0]&0x08 != 0
			anchoCanvas = 1 + int(chunk[4]) + int(chunk[5])<<8 + int(chunk[6])<<16
			altoCanvas = 1 + int(chunk[7]) + int(chunk[8])<<8 + int(chunk[9])<<16
			if anchoCanvas > ports.DimensionMaximaFotoImagen || altoCanvas > ports.DimensionMaximaFotoImagen || int64(anchoCanvas)*int64(altoCanvas) > ports.PixelesMaximosFotoImagen {
				return 0, errFormato
			}
		case "VP8L":
			if imagenVista {
				return 0, errFormato
			}
			config, err := vp8l.DecodeConfig(bytes.NewReader(chunk))
			if err != nil || !dimensionesWebPValidas(config.Width, config.Height, extendido, anchoCanvas, altoCanvas) {
				return 0, errFormato
			}
			imagenVista = true
		case "VP8 ":
			if imagenVista {
				return 0, errFormato
			}
			decoder := vp8.NewDecoder()
			decoder.Init(bytes.NewReader(chunk), len(chunk))
			cabecera, err := decoder.DecodeFrameHeader()
			if err != nil || !dimensionesWebPValidas(cabecera.Width, cabecera.Height, extendido, anchoCanvas, altoCanvas) {
				return 0, errFormato
			}
			imagenVista = true
		case "EXIF":
			if exifVisto || !extendido || !exifEsperado {
				return 0, errFormato
			}
			var err error
			orientacion, err = orientacionTIFF(chunk)
			if err != nil {
				return 0, err
			}
			exifVisto = true
		case "ANIM", "ANMF":
			return 0, errFormato
		}
		i += 8 + int(longitud) + int(longitud%2)
	}
	if !imagenVista || exifEsperado != exifVisto {
		return 0, errFormato
	}
	return orientacion, nil
}

func dimensionesWebPValidas(ancho, alto int, extendido bool, anchoCanvas, altoCanvas int) bool {
	return ancho > 0 && alto > 0 && ancho <= ports.DimensionMaximaFotoImagen && alto <= ports.DimensionMaximaFotoImagen && int64(ancho)*int64(alto) <= ports.PixelesMaximosFotoImagen && (!extendido || ancho == anchoCanvas && alto == altoCanvas)
}

func orientacionPNG(b []byte) (uint8, error) {
	orientacion := uint8(1)
	exifVisto, ihdrVisto, finVisto := false, false, false
	for i := 8; i < len(b); {
		if len(b)-i < 12 {
			return 0, errFormato
		}
		longitud := uint64(binary.BigEndian.Uint32(b[i : i+4]))
		if longitud > uint64(len(b)-i-12) {
			return 0, errFormato
		}
		final := i + 12 + int(longitud)
		if crc32.ChecksumIEEE(b[i+4:final-4]) != binary.BigEndian.Uint32(b[final-4:final]) {
			return 0, errFormato
		}
		chunk := b[i+8 : final-4]
		switch string(b[i+4 : i+8]) {
		case "IHDR":
			if ihdrVisto || i != 8 || len(chunk) != 13 {
				return 0, errFormato
			}
			ihdrVisto = true
		case "eXIf":
			if exifVisto {
				return 0, errFormato
			}
			var err error
			orientacion, err = orientacionTIFF(chunk)
			if err != nil {
				return 0, err
			}
			exifVisto = true
		case "IEND":
			if len(chunk) != 0 || final != len(b) {
				return 0, errFormato
			}
			finVisto = true
		}
		i = final
	}
	if !ihdrVisto || !finVisto {
		return 0, errFormato
	}
	return orientacion, nil
}

func orientacionTIFF(b []byte) (uint8, error) {
	if bytes.HasPrefix(b, []byte("Exif\x00\x00")) {
		b = b[6:]
	}
	return leerOrientacionEXIF(b)
}
