package postgrespublico

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	httpbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/httpapi"
)

const (
	maximoPosicionesB10 = 100000
	// maximoPaginaB10 cubre el límite máximo de la consulta pública (100)
	// más la fila que indica si hay más.
	maximoPaginaB10 = 101
)

var (
	patronReferenciaBolsaB10 = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{2,159}$`)
	patronDocumentoB10       = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)
	estadosB10               = map[string]struct{}{
		"disponible": {}, "ocupado": {}, "no_disponible": {},
		"excluido": {}, "renuncia_pendiente": {},
	}
)

var (
	_ httpbolsa.FuenteBolsasPublicas       = (*Fuente)(nil)
	_ httpbolsa.FuentePaginaBolsasPublicas = (*Fuente)(nil)
)

// BolsasPublicas devuelve exclusivamente la proyeccion pública B10. La
// instantánea repetible se ancla al manifiesto que ya validó esta Fuente.
func (f *Fuente) BolsasPublicas(ctx context.Context) ([]httpbolsa.BolsaPublica, time.Time, error) {
	if ctx == nil || !f.valida() {
		return nil, time.Time{}, ErrPostgreSQLPublicoNoDisponible
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return nil, time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)

	generadoEn, err := leerInstanteFuenteB10(ctx, tx, f.manifiestoSHA256)
	if err != nil {
		return nil, time.Time{}, err
	}
	bolsas, err := leerBolsasB10(ctx, tx)
	if err != nil {
		return nil, time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	return bolsas, generadoEn, nil
}

// ListaPublica conserva el contrato B10 vigente. El manejador aplica su
// paginación y filtro sobre esta secuencia ya minimizada; la consulta no lee
// nombres, contactos ni referencias de participación.
func (f *Fuente) ListaPublica(ctx context.Context, bolsaRef string) (httpbolsa.BolsaPublica, []httpbolsa.PosicionPublica, time.Time, error) {
	if ctx == nil || !f.valida() || bolsaRef != strings.TrimSpace(bolsaRef) || !patronReferenciaBolsaB10.MatchString(bolsaRef) {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, ErrPostgreSQLPublicoNoDisponible
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)

	generadoEn, err := leerInstanteFuenteB10(ctx, tx, f.manifiestoSHA256)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	bolsa, encontrada, err := leerBolsaB10(ctx, tx, bolsaRef)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	if !encontrada {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, httpbolsa.ErrBolsaPublicaNoEncontrada
	}
	posiciones, err := leerPosicionesB10(ctx, tx, bolsaRef, bolsa.Total)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	return bolsa, posiciones, generadoEn, nil
}

// PaginaListaPublica lee solo el tramo pedido de la lista B10: como mucho
// cantidad posiciones a partir de desde. Antes coteja en la misma instantánea
// que la lista completa sigue siendo exactamente 1..total (recuento, mínimo y
// máximo sobre la clave primaria), así que conserva la garantía de integridad
// de ListaPublica sin trasladar miles de filas por petición.
func (f *Fuente) PaginaListaPublica(ctx context.Context, bolsaRef string, desde, cantidad int) (httpbolsa.BolsaPublica, []httpbolsa.PosicionPublica, time.Time, error) {
	if ctx == nil || !f.valida() || bolsaRef != strings.TrimSpace(bolsaRef) || !patronReferenciaBolsaB10.MatchString(bolsaRef) ||
		desde < 1 || desde > maximoPosicionesB10 || cantidad < 1 || cantidad > maximoPaginaB10 {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, ErrPostgreSQLPublicoNoDisponible
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	defer rollbackPostgreSQLPublico(tx)

	generadoEn, err := leerInstanteFuenteB10(ctx, tx, f.manifiestoSHA256)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	bolsa, encontrada, err := leerBolsaB10(ctx, tx, bolsaRef)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	if !encontrada {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, httpbolsa.ErrBolsaPublicaNoEncontrada
	}
	if err := comprobarSecuenciaPosicionesB10(ctx, tx, bolsaRef, bolsa.Total); err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	posiciones, err := leerTramoPosicionesB10(ctx, tx, bolsaRef, bolsa.Total, desde, cantidad)
	if err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return httpbolsa.BolsaPublica{}, nil, time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	return bolsa, posiciones, generadoEn, nil
}

// comprobarSecuenciaPosicionesB10 exige que las posiciones de la bolsa sean
// exactamente 1..total. La clave primaria (bolsa_ref, orden) impide órdenes
// repetidos, así que recuento, mínimo y máximo bastan para probarlo. Lo hace
// PostgreSQL con un recorrido solo de índice, sin enviar las filas.
func comprobarSecuenciaPosicionesB10(ctx context.Context, tx pgx.Tx, bolsaRef string, total int) error {
	var recuento, minimo, maximo int64
	if err := tx.QueryRow(ctx, `
		SELECT count(*), COALESCE(min(orden), 0), COALESCE(max(orden), 0)
		  FROM vec_bolsa_publica_lectura.posiciones_bolsa_v1
		 WHERE bolsa_ref = $1`, bolsaRef).Scan(&recuento, &minimo, &maximo); err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	esperado := int64(total)
	if recuento != esperado || maximo != esperado || (total == 0 && minimo != 0) || (total > 0 && minimo != 1) {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	return nil
}

// leerTramoPosicionesB10 acota también el orden por arriba: la vista es
// security_barrier y PostgreSQL no puede usar el orden del índice con LIMIT,
// pero con el intervalo cerrado solo recorre las filas del tramo.
func leerTramoPosicionesB10(ctx context.Context, tx pgx.Tx, bolsaRef string, total, desde, cantidad int) ([]httpbolsa.PosicionPublica, error) {
	esperadas := 0
	if desde <= total {
		esperadas = min(cantidad, total-desde+1)
	}
	filas, err := tx.Query(ctx, `
		SELECT orden, documento_enmascarado, estado_clave
		  FROM vec_bolsa_publica_lectura.posiciones_bolsa_v1
		 WHERE bolsa_ref = $1 AND orden >= $2 AND orden < $3
		 ORDER BY orden`, bolsaRef, int64(desde), int64(desde)+int64(cantidad))
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	posiciones := make([]httpbolsa.PosicionPublica, 0, esperadas)
	for filas.Next() {
		var orden int64
		var posicion httpbolsa.PosicionPublica
		if err := filas.Scan(&orden, &posicion.DocumentoEnmascarado, &posicion.EstadoClave); err != nil ||
			orden != int64(desde+len(posiciones)) || orden > int64(total) ||
			!patronDocumentoB10.MatchString(posicion.DocumentoEnmascarado) || !estadoB10Valido(posicion.EstadoClave) {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		posicion.Orden = int(orden)
		posiciones = append(posiciones, posicion)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(posiciones) != esperadas {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return posiciones, nil
}

func leerInstanteFuenteB10(ctx context.Context, tx pgx.Tx, manifiestoEsperado string) (time.Time, error) {
	var actualizadaEn time.Time
	var manifiesto string
	if err := tx.QueryRow(ctx, `
		SELECT actualizada_en, manifiesto_sha256
		  FROM vec_bolsa_publica_lectura.fuente_publica_v2
		 WHERE control_id IS TRUE`).Scan(&actualizadaEn, &manifiesto); err != nil {
		return time.Time{}, errorPostgreSQLPublico(ctx, err)
	}
	if actualizadaEn.IsZero() || !huellasIguales(manifiesto, manifiestoEsperado) {
		return time.Time{}, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return instanteUTC(actualizadaEn), nil
}

func leerBolsasB10(ctx context.Context, tx pgx.Tx) ([]httpbolsa.BolsaPublica, error) {
	filas, err := tx.Query(ctx, `
		SELECT bolsa_ref, categoria, grupos, tipo_lista, vigente_desde, vigente_hasta, total
		  FROM vec_bolsa_publica_lectura.bolsas_v1
		 WHERE vigente_desde <= CURRENT_TIMESTAMP
		   AND (vigente_hasta IS NULL OR vigente_hasta >= CURRENT_TIMESTAMP)
		 ORDER BY bolsa_ref`)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	bolsas := make([]httpbolsa.BolsaPublica, 0)
	for filas.Next() {
		bolsa, err := escanearBolsaB10(filas)
		if err != nil {
			return nil, err
		}
		bolsas = append(bolsas, bolsa)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	return bolsas, nil
}

func leerBolsaB10(ctx context.Context, tx pgx.Tx, bolsaRef string) (httpbolsa.BolsaPublica, bool, error) {
	filas, err := tx.Query(ctx, `
		SELECT bolsa_ref, categoria, grupos, tipo_lista, vigente_desde, vigente_hasta, total
		  FROM vec_bolsa_publica_lectura.bolsas_v1
		 WHERE bolsa_ref = $1
		   AND vigente_desde <= CURRENT_TIMESTAMP
		   AND (vigente_hasta IS NULL OR vigente_hasta >= CURRENT_TIMESTAMP)`, bolsaRef)
	if err != nil {
		return httpbolsa.BolsaPublica{}, false, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	if !filas.Next() {
		if err := filas.Err(); err != nil {
			return httpbolsa.BolsaPublica{}, false, errorPostgreSQLPublico(ctx, err)
		}
		return httpbolsa.BolsaPublica{}, false, nil
	}
	bolsa, err := escanearBolsaB10(filas)
	if err != nil {
		return httpbolsa.BolsaPublica{}, false, err
	}
	if filas.Next() || filas.Err() != nil {
		return httpbolsa.BolsaPublica{}, false, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return bolsa, true, nil
}

func escanearBolsaB10(filas pgx.Rows) (httpbolsa.BolsaPublica, error) {
	var bolsa httpbolsa.BolsaPublica
	var total int64
	if err := filas.Scan(&bolsa.BolsaRef, &bolsa.Categoria, &bolsa.Grupos, &bolsa.TipoLista, &bolsa.VigenteDesde, &bolsa.VigenteHasta, &total); err != nil {
		return httpbolsa.BolsaPublica{}, ErrDatosPostgreSQLPublicosNoConfiables
	}
	if total < 0 || total > maximoPosicionesB10 || total > math.MaxInt || !bolsaB10Valida(bolsa) {
		return httpbolsa.BolsaPublica{}, ErrDatosPostgreSQLPublicosNoConfiables
	}
	bolsa.Total = int(total)
	bolsa.VigenteDesde = instanteUTC(bolsa.VigenteDesde)
	if bolsa.VigenteHasta != nil {
		instante := instanteUTC(*bolsa.VigenteHasta)
		bolsa.VigenteHasta = &instante
	}
	return bolsa, nil
}

func leerPosicionesB10(ctx context.Context, tx pgx.Tx, bolsaRef string, total int) ([]httpbolsa.PosicionPublica, error) {
	filas, err := tx.Query(ctx, `
		SELECT orden, documento_enmascarado, estado_clave
		  FROM vec_bolsa_publica_lectura.posiciones_bolsa_v1
		 WHERE bolsa_ref = $1
		 ORDER BY orden`, bolsaRef)
	if err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	defer filas.Close()
	posiciones := make([]httpbolsa.PosicionPublica, 0, total)
	for filas.Next() {
		var orden int64
		var posicion httpbolsa.PosicionPublica
		if err := filas.Scan(&orden, &posicion.DocumentoEnmascarado, &posicion.EstadoClave); err != nil ||
			orden < 1 || orden > maximoPosicionesB10 || orden > math.MaxInt ||
			!patronDocumentoB10.MatchString(posicion.DocumentoEnmascarado) || !estadoB10Valido(posicion.EstadoClave) ||
			int(orden) != len(posiciones)+1 {
			return nil, ErrDatosPostgreSQLPublicosNoConfiables
		}
		posicion.Orden = int(orden)
		posiciones = append(posiciones, posicion)
	}
	if err := filas.Err(); err != nil {
		return nil, errorPostgreSQLPublico(ctx, err)
	}
	if len(posiciones) != total {
		return nil, ErrDatosPostgreSQLPublicosNoConfiables
	}
	return posiciones, nil
}

func bolsaB10Valida(bolsa httpbolsa.BolsaPublica) bool {
	if !patronReferenciaBolsaB10.MatchString(bolsa.BolsaRef) || strings.TrimSpace(bolsa.Categoria) == "" ||
		strings.TrimSpace(bolsa.TipoLista) == "" || bolsa.VigenteDesde.IsZero() {
		return false
	}
	for _, grupo := range bolsa.Grupos {
		if strings.TrimSpace(grupo) == "" {
			return false
		}
	}
	return bolsa.VigenteHasta == nil || !bolsa.VigenteHasta.Before(bolsa.VigenteDesde)
}

func estadoB10Valido(estado string) bool {
	_, valido := estadosB10[estado]
	return valido
}
