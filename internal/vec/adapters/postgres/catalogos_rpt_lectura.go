package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const maximoDocumentoCanonicoRPT = 16 << 20

var huellaRPT = regexp.MustCompile(`^[0-9a-f]{64}$`)
var claveRPT = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

type publicacionRPTWire struct {
	CatalogoID        string    `json:"catalogo_id"`
	Version           int       `json:"version"`
	HuellaSHA256      string    `json:"huella_sha256"`
	DocumentoCanonico string    `json:"documento_canonico"`
	PublicadaEn       time.Time `json:"publicada_en"`
}

func (p publicacionRPTWire) referencia() ports.ReferenciaPublicacionRPT {
	return ports.ReferenciaPublicacionRPT{CatalogoID: p.CatalogoID, Version: p.Version, HuellaSHA256: p.HuellaSHA256}
}

// comprobarPublicacionRPT verifica los bytes originales, el documento de
// dominio y cada entrada antes de entregar una publicación al consumidor.
// La huella nunca se reconstruye desde JSONB de una fila de control.
func comprobarPublicacionRPT(p publicacionRPTWire) (ports.PublicacionRPT, map[string]domain.EntradaCatalogoConfigurable, error) {
	var cero ports.PublicacionRPT
	if p.CatalogoID == "" || p.Version < 1 || !huellaRPT.MatchString(p.HuellaSHA256) ||
		len(p.DocumentoCanonico) < 2 || len(p.DocumentoCanonico) > maximoDocumentoCanonicoRPT {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	suma := sha256.Sum256([]byte(p.DocumentoCanonico))
	if hex.EncodeToString(suma[:]) != p.HuellaSHA256 {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	var catalogo domain.CatalogoConfigurable
	if decodificarRPT([]byte(p.DocumentoCanonico), &catalogo) != nil ||
		catalogo.ID != p.CatalogoID || catalogo.Version != p.Version ||
		catalogo.Estado != domain.EstadoCatalogoPublicado || catalogo.Validar() != nil {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	representacion, err := json.Marshal(canonico)
	if err != nil || !bytes.Equal(representacion, []byte(p.DocumentoCanonico)) {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	entradas := make(map[string]domain.EntradaCatalogoConfigurable, len(catalogo.Entradas))
	for _, entrada := range catalogo.Entradas {
		entradas[entrada.Clave] = entrada
	}
	return ports.PublicacionRPT{Referencia: p.referencia(), DocumentoCanonico: p.DocumentoCanonico, PublicadaEn: p.PublicadaEn}, entradas, nil
}

type categoriaRPTWire struct {
	CategoriaID  string          `json:"categoria_id"`
	CatalogoID   string          `json:"catalogo_id"`
	Version      int             `json:"version"`
	HuellaSHA256 string          `json:"huella_sha256"`
	Revision     int64           `json:"revision"`
	Estado       string          `json:"estado"`
	Etiqueta     string          `json:"etiqueta"`
	Definicion   json.RawMessage `json:"definicion"`
}

func (c categoriaRPTWire) referencia() ports.ReferenciaPublicacionRPT {
	return ports.ReferenciaPublicacionRPT{CatalogoID: c.CatalogoID, Version: c.Version, HuellaSHA256: c.HuellaSHA256}
}

type listaRPTWire struct {
	Items           []json.RawMessage `json:"items"`
	Publicaciones   []json.RawMessage `json:"publicaciones"`
	HayMas          bool              `json:"hay_mas"`
	SiguienteCursor *string           `json:"siguiente_cursor"`
}

func decodificarListaRPT(datos []byte, consulta ports.ConsultaCategoriasHabilitadasRPT) (ports.ResultadoCategoriasHabilitadasRPT, error) {
	var resultado ports.ResultadoCategoriasHabilitadasRPT
	var wire listaRPTWire
	if numeroClavesRPT(datos) != 4 || decodificarRPT(datos, &wire) != nil ||
		len(wire.Items) > consulta.Limite || len(wire.Publicaciones) > len(wire.Items) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	type publicacionCompleta struct {
		publicacion ports.PublicacionRPT
		entradas    map[string]domain.EntradaCatalogoConfigurable
	}
	publicaciones := make(map[ports.ReferenciaPublicacionRPT]publicacionCompleta, len(wire.Publicaciones))
	for _, bruto := range wire.Publicaciones {
		var p publicacionRPTWire
		if numeroClavesRPT(bruto) != 4 || decodificarRPT(bruto, &p) != nil {
			return resultado, ports.ErrLecturaRPTNoConfiable
		}
		publicacion, entradas, err := comprobarPublicacionRPT(p)
		if err != nil || p.CatalogoID != consulta.CatalogoID {
			return resultado, ports.ErrLecturaRPTNoConfiable
		}
		if _, duplicada := publicaciones[p.referencia()]; duplicada {
			return resultado, ports.ErrLecturaRPTNoConfiable
		}
		publicaciones[p.referencia()] = publicacionCompleta{publicacion: publicacion, entradas: entradas}
		resultado.Publicaciones = append(resultado.Publicaciones, publicacion)
	}
	anterior := consulta.CursorCategoriaID
	for _, bruto := range wire.Items {
		var c categoriaRPTWire
		if numeroClavesRPT(bruto) != 8 || decodificarRPT(bruto, &c) != nil ||
			c.CatalogoID != consulta.CatalogoID || c.CategoriaID <= anterior || c.Revision < 1 ||
			c.Estado != "habilitada" {
			return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
		}
		p, existe := publicaciones[c.referencia()]
		entrada, publicada := p.entradas[c.CategoriaID]
		var recibida domain.EntradaCatalogoConfigurable
		if !existe || !publicada || decodificarRPT(c.Definicion, &recibida) != nil ||
			!reflect.DeepEqual(recibida, entrada) || c.Etiqueta != entrada.Etiqueta {
			return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
		}
		resultado.Categorias = append(resultado.Categorias, ports.CategoriaHabilitadaRPT{
			CategoriaID: c.CategoriaID, Publicacion: c.referencia(), Revision: c.Revision,
			Estado: c.Estado, Etiqueta: c.Etiqueta, Definicion: entrada,
		})
		anterior = c.CategoriaID
	}
	if wire.HayMas {
		if len(resultado.Categorias) == 0 || wire.SiguienteCursor == nil || *wire.SiguienteCursor != anterior {
			return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
		}
	} else if wire.SiguienteCursor != nil {
		return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
	}
	resultado.Encontrado, resultado.HayMas, resultado.SiguienteCursor = true, wire.HayMas, wire.SiguienteCursor
	return resultado, nil
}

type publicacionHistoricaRPTWire struct {
	Publicacion   json.RawMessage `json:"publicacion"`
	Entrada       json.RawMessage `json:"entrada"`
	ControlActual json.RawMessage `json:"control_actual"`
}

type controlActualRPTWire struct {
	CatalogoID   string `json:"catalogo_id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Revision     int64  `json:"revision"`
	Estado       string `json:"estado"`
}

func decodificarPublicacionHistoricaRPT(datos []byte, consulta ports.ConsultaPublicacionCategoriaRPT) (ports.ResultadoPublicacionCategoriaRPT, error) {
	var resultado ports.ResultadoPublicacionCategoriaRPT
	var wire publicacionHistoricaRPTWire
	if numeroClavesRPT(datos) != 3 || decodificarRPT(datos, &wire) != nil ||
		numeroClavesRPT(wire.Publicacion) != 5 {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	var p publicacionRPTWire
	if decodificarRPT(wire.Publicacion, &p) != nil || p.referencia() != consulta.Referencia ||
		!instanteRPTValido(p.PublicadaEn) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	publicacion, entradas, err := comprobarPublicacionRPT(p)
	if err != nil {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	var entrada domain.EntradaCatalogoConfigurable
	publicada, existe := entradas[consulta.CategoriaID]
	if !existe || decodificarRPT(wire.Entrada, &entrada) != nil ||
		!reflect.DeepEqual(entrada, publicada) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	resultado.Encontrado, resultado.Publicacion, resultado.Entrada = true, &publicacion, &publicada
	if !bytes.Equal(bytes.TrimSpace(wire.ControlActual), []byte("null")) {
		var c controlActualRPTWire
		if numeroClavesRPT(wire.ControlActual) != 5 || decodificarRPT(wire.ControlActual, &c) != nil ||
			c.CatalogoID != consulta.Referencia.CatalogoID || c.Version < 1 ||
			!huellaRPT.MatchString(c.HuellaSHA256) || c.Revision < 1 ||
			(c.Estado != "habilitada" && c.Estado != "deshabilitada" && c.Estado != "tombstone") {
			return ports.ResultadoPublicacionCategoriaRPT{}, ports.ErrLecturaRPTNoConfiable
		}
		resultado.ControlActual = &ports.ControlCategoriaRPT{
			Publicacion: ports.ReferenciaPublicacionRPT{CatalogoID: c.CatalogoID, Version: c.Version, HuellaSHA256: c.HuellaSHA256},
			Revision:    c.Revision, Estado: c.Estado,
		}
	}
	return resultado, nil
}

type usoCategoriaRPTWire struct {
	Consumidor        string     `json:"consumidor"`
	UsoRef            string     `json:"uso_ref"`
	CategoriaID       string     `json:"categoria_id"`
	CatalogoID        string     `json:"catalogo_id"`
	Version           int        `json:"version"`
	HuellaSHA256      string     `json:"huella_sha256"`
	Estado            string     `json:"estado"`
	Revision          int64      `json:"revision"`
	ReservaReciboRef  string     `json:"reserva_recibo_ref"`
	TerminalReciboRef *string    `json:"terminal_recibo_ref"`
	ReservadoEn       time.Time  `json:"reservado_en"`
	TerminalEn        *time.Time `json:"terminal_en"`
}

func decodificarUsoCategoriaRPT(datos []byte, consulta ports.ConsultaUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	var resultado ports.ResultadoUsoCategoriaRPT
	var u usoCategoriaRPTWire
	if numeroClavesRPT(datos) != 12 || decodificarRPT(datos, &u) != nil ||
		u.Consumidor != consulta.Consumidor || u.UsoRef != consulta.UsoRef ||
		u.ReservaReciboRef != consulta.ReservaReciboRef || !claveRPT.MatchString(u.CategoriaID) ||
		!claveRPT.MatchString(u.CatalogoID) || u.Version < 1 || !huellaRPT.MatchString(u.HuellaSHA256) ||
		!instanteRPTValido(u.ReservadoEn) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	terminal := u.Estado == "confirmado" || u.Estado == "cancelado"
	if !(u.Estado == "reservado" || terminal) ||
		(!terminal && (u.Revision != 1 || u.TerminalReciboRef != nil || u.TerminalEn != nil)) ||
		(terminal && (u.Revision != 2 || u.TerminalReciboRef == nil || *u.TerminalReciboRef == "" ||
			u.TerminalEn == nil || !instanteRPTValido(*u.TerminalEn) || u.TerminalEn.Before(u.ReservadoEn))) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	resultado.Encontrado = true
	resultado.Uso = &ports.UsoCategoriaRPT{
		Consumidor: u.Consumidor, UsoRef: u.UsoRef, CategoriaID: u.CategoriaID,
		Publicacion: ports.ReferenciaPublicacionRPT{CatalogoID: u.CatalogoID, Version: u.Version, HuellaSHA256: u.HuellaSHA256},
		Estado:      u.Estado, Revision: u.Revision, ReservaReciboRef: u.ReservaReciboRef,
		TerminalReciboRef: u.TerminalReciboRef, ReservadoEn: u.ReservadoEn.UTC(), TerminalEn: u.TerminalEn,
	}
	return resultado, nil
}

func instanteRPTValido(instante time.Time) bool {
	_, desfase := instante.Zone()
	return !instante.IsZero() && desfase == 0 && instante.Nanosecond()%1_000 == 0
}

func numeroClavesRPT(bruto []byte) int {
	var campos map[string]json.RawMessage
	if json.Unmarshal(bruto, &campos) != nil {
		return -1
	}
	return len(campos)
}

func decodificarRPT(bruto []byte, destino any) error {
	decoder := json.NewDecoder(bytes.NewReader(bruto))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destino); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return ports.ErrLecturaRPTNoConfiable
	}
	return nil
}
