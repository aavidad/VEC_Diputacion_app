package informemovimientos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type Preparador struct {
	renderer vecports.RenderizadorDocumento
	catalogo Catalogo
	version  int64
	huella   string
}

// DocumentoEjemplo todavía no es una descarga autorizada ni un recibo.
type DocumentoEjemplo struct {
	Contenido       []byte
	CatalogoRef     string
	CatalogoVersion int64
	CatalogoSHA256  string
}

func Nuevo(renderer vecports.RenderizadorDocumento, datos io.Reader) (*Preparador, error) {
	if nulo(renderer) || nulo(datos) || renderer.Formato() != vecdomain.FormatoDocumentoPDF {
		return nil, ErrEjemploNoDisponible
	}
	raw, err := io.ReadAll(io.LimitReader(datos, 65537))
	if err != nil || len(raw) > 65536 {
		return nil, ErrEjemploInvalido
	}
	var c Catalogo
	if err := leerJSON(bytes.NewReader(raw), &c); err != nil {
		return nil, err
	}
	version, err := validarCatalogo(c)
	if err != nil {
		return nil, err
	}
	suma := sha256.Sum256(raw)
	return &Preparador{renderer: renderer, catalogo: c, version: version, huella: hex.EncodeToString(suma[:])}, nil
}

// PrepararEjemploSintetico lista hechos. No interpreta pares de fichajes, horas
// trabajadas, presencia, ausencia ni derechos. No implementa un puerto nominal.
func (p *Preparador) PrepararEjemploSintetico(ctx context.Context, e EjemploSintetico) (DocumentoEjemplo, error) {
	cero := DocumentoEjemplo{}
	if p == nil || nulo(p.renderer) || ctx == nil {
		return cero, ErrEjemploNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	zona, err := time.LoadLocation(e.ZonaHoraria)
	if err != nil || !texto(e.ZonaHoraria, 64) || e.ZonaHoraria == "Local" || !e.Demo || !texto(e.Nombre, 128) || e.Completo == nil || e.Marcajes == nil || len(e.Marcajes) > 10000 {
		return cero, ErrEjemploInvalido
	}
	desde, hasta, err := periodo(e.Periodo, zona)
	if err != nil {
		return cero, err
	}
	fin := hasta.AddDate(0, 0, 1)
	ordenados := append([]ports.MarcajeDia(nil), e.Marcajes...)
	for _, m := range ordenados {
		if m.InstanteUTC.Location() != time.UTC || m.InstanteUTC.IsZero() || m.InstanteUTC.Nanosecond()%1000 != 0 || m.InstanteUTC.Before(desde) || !m.InstanteUTC.Before(fin) || p.catalogo.Movimientos[string(m.Movimiento)] == "" {
			return cero, ErrEjemploInvalido
		}
		if m.Origen != nil && p.catalogo.Origenes[*m.Origen] == "" {
			return cero, ErrEjemploInvalido
		}
	}
	sort.SliceStable(ordenados, func(i, j int) bool { return ordenados[i].InstanteUTC.Before(ordenados[j].InstanteUTC) })
	c := p.catalogo
	fuente := c.FuenteIncompleta
	if *e.Completo {
		fuente = c.FuenteCompleta
	}
	parrafos := []string{c.Sintetico, strings.NewReplacer("{{nombre}}", e.Nombre).Replace(c.NombreSintetico), strings.NewReplacer("{{desde}}", desde.Format(c.FormatoFecha), "{{hasta}}", hasta.Format(c.FormatoFecha)).Replace(c.Periodo), strings.NewReplacer("{{zona}}", e.ZonaHoraria).Replace(c.Zona), fuente}
	for _, m := range ordenados {
		if err := ctx.Err(); err != nil {
			return cero, err
		}
		origen := c.OrigenSinVerificar
		if m.Origen != nil {
			origen = c.Origenes[*m.Origen]
		}
		at := m.InstanteUTC.In(zona)
		parrafos = append(parrafos, strings.NewReplacer("{{fecha}}", at.Format(c.FormatoFecha), "{{hora}}", at.Format(c.FormatoHora), "{{movimiento}}", c.Movimientos[string(m.Movimiento)], "{{origen}}", origen).Replace(c.Fila))
	}
	if len(ordenados) == 0 {
		parrafos = append(parrafos, c.Vacio)
	}
	parrafos = append(parrafos, c.Limite)
	contenido, err := p.renderer.Renderizar(ctx, vecdomain.ContenidoDocumento{Titulo: c.Titulo, Parrafos: parrafos})
	if err != nil {
		return cero, err
	}
	// El mismo límite documental de los informes de saldo y permisos existentes.
	if len(contenido) == 0 || len(contenido) > 2*1024*1024 {
		return cero, ErrEjemploInvalido
	}
	contenido = append([]byte(nil), contenido...)
	if err := p.renderer.ValidarSalida(ctx, contenido); err != nil {
		return cero, err
	}
	if !bytes.Contains(contenido, []byte("/Lang ("+c.Idioma+")")) {
		return cero, ErrEjemploNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return DocumentoEjemplo{Contenido: contenido, CatalogoRef: c.Referencia, CatalogoVersion: p.version, CatalogoSHA256: p.huella}, nil
}

// Conserva los periodos y sus límites civiles de la consulta de saldo existente.
func periodo(p ports.PeriodoConsultaSaldo, zona *time.Location) (time.Time, time.Time, error) {
	desde, e1 := time.ParseInLocation("2006-01-02", p.Desde, zona)
	hasta, e2 := time.ParseInLocation("2006-01-02", p.Hasta, zona)
	if e1 != nil || e2 != nil || desde.Format("2006-01-02") != p.Desde || hasta.Format("2006-01-02") != p.Hasta || hasta.Before(desde) || hasta.After(desde.AddDate(1, 0, 0)) {
		return time.Time{}, time.Time{}, ErrEjemploInvalido
	}
	valido := false
	switch p.Tipo {
	case ports.PeriodoSaldoHoy:
		valido = desde.Equal(hasta)
	case ports.PeriodoSaldoSemana:
		valido = desde.Weekday() == time.Monday && hasta.Equal(desde.AddDate(0, 0, 6))
	case ports.PeriodoSaldoMes:
		valido = desde.Day() == 1 && hasta.Equal(desde.AddDate(0, 1, -1))
	case ports.PeriodoSaldoAnio:
		valido = desde.Month() == time.January && desde.Day() == 1 && hasta.Equal(desde.AddDate(1, 0, -1))
	case ports.PeriodoSaldoRango:
		valido = true
	}
	if !valido {
		return time.Time{}, time.Time{}, ErrEjemploInvalido
	}
	return desde, hasta, nil
}
