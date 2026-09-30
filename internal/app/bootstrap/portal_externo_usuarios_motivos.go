package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	usuarios "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// El catálogo es el que AUT17 admite. Su contenido se recibe como datos,
// nunca desde una petición del portal ni desde la identidad del candidato.
const catalogoMotivosUsuariosExterno = "motivos_usuarios_propios_desarrollo"

var errMotivosUsuariosExterno = errors.New("bootstrap: motivos propios de Usuarios no disponibles")

type EntradaMotivosUsuariosExterno struct {
	Tipo      string            `json:"tipo"`
	Clave     string            `json:"clave"`
	ModuloID  string            `json:"modulo_id"`
	Acciones  []string          `json:"acciones"`
	Etiquetas map[string]string `json:"etiquetas"`
}

type CatalogoMotivosUsuariosExterno struct {
	CatalogoID  string                          `json:"catalogo_id"`
	Version     int                             `json:"version"`
	PublicadoEn time.Time                       `json:"publicado_en"`
	Entradas    []EntradaMotivosUsuariosExterno `json:"entradas"`
}

type SolicitudMotivosUsuariosExterno struct {
	AprobacionRef     string                         `json:"aprobacion_ref"`
	SecuenciaEsperada int64                          `json:"secuencia_esperada"`
	Catalogo          CatalogoMotivosUsuariosExterno `json:"catalogo"`
}

func (SolicitudMotivosUsuariosExterno) String() string { return "[PLAN DE MOTIVOS USUARIOS]" }

func claveEntradaMotivosUsuariosExterno(tipo string) string {
	return referenciaAltaContratacionTemporalDesarrollo("motivo_", catalogoMotivosUsuariosExterno+"\x00"+tipo)
}

func accionesEntradaMotivosUsuariosExterno(tipo string) []string {
	switch tipo {
	case "consulta":
		return []string{usuarios.AccionConsultarPreferencias, usuarios.AccionConsultarImagen, usuarios.AccionConsultarCorreos}
	case "actualizacion":
		return []string{usuarios.AccionActualizarPreferencias, usuarios.AccionActualizarImagen, usuarios.AccionAnadirCorreo,
			usuarios.AccionReenviarCorreo, usuarios.AccionVerificarCorreo, usuarios.AccionActivarCorreo, usuarios.AccionRetirarCorreo}
	}
	return nil
}

func LeerSolicitudMotivosUsuariosExterno(b []byte) (SolicitudMotivosUsuariosExterno, error) {
	var s SolicitudMotivosUsuariosExterno
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var sobra any
	if len(b) == 0 || len(b) > 262144 || validarClavesJSONUnicas(b) != nil || d.Decode(&s) != nil || d.Decode(&sobra) != io.EOF {
		return s, errMotivosUsuariosExterno
	}
	if _, err := s.HuellaSHA256(); err != nil {
		return SolicitudMotivosUsuariosExterno{}, err
	}
	return s, nil
}

func (s SolicitudMotivosUsuariosExterno) validar() error {
	c := s.Catalogo
	if !textoProvisionUsuariosValido(s.AprobacionRef) || s.SecuenciaEsperada < 0 || s.SecuenciaEsperada == math.MaxInt64 ||
		c.CatalogoID != catalogoMotivosUsuariosExterno || c.Version < 1 || c.Version > math.MaxInt32 ||
		c.PublicadoEn.IsZero() || c.PublicadoEn.Location() != time.UTC || c.PublicadoEn.Year() < 1 || c.PublicadoEn.Year() > 9999 ||
		c.PublicadoEn.Nanosecond()%1000 != 0 || len(c.Entradas) != 2 {
		return errMotivosUsuariosExterno
	}
	for i, tipo := range []string{"consulta", "actualizacion"} {
		e := c.Entradas[i]
		if e.Tipo != tipo || e.Clave != claveEntradaMotivosUsuariosExterno(tipo) || e.ModuloID != "usuarios" ||
			!reflect.DeepEqual(e.Acciones, accionesEntradaMotivosUsuariosExterno(tipo)) || len(e.Etiquetas) == 0 || len(e.Etiquetas) > 32 {
			return errMotivosUsuariosExterno
		}
		for idioma, etiqueta := range e.Etiquetas {
			if len(idioma) < 2 || len(idioma) > 16 || len(etiqueta) == 0 || len(etiqueta) > 512 {
				return errMotivosUsuariosExterno
			}
		}
	}
	return nil
}

func (s SolicitudMotivosUsuariosExterno) HuellaSHA256() (string, error) {
	if err := s.validar(); err != nil {
		return "", err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", errMotivosUsuariosExterno
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

type ResumenMotivosUsuariosExterno struct {
	Estado              string                         `json:"estado"`
	HuellaPlan          string                         `json:"plan_huella_sha256"`
	Secuencia           int64                          `json:"secuencia"`
	EventoRef           string                         `json:"evento_ref"`
	MotivoConsulta      core.ReferenciaEntradaCatalogo `json:"motivo_consulta"`
	MotivoActualizacion core.ReferenciaEntradaCatalogo `json:"motivo_actualizacion"`
}

func (s SolicitudMotivosUsuariosExterno) Resumen() (ResumenMotivosUsuariosExterno, error) {
	h, err := s.HuellaSHA256()
	if err != nil {
		return ResumenMotivosUsuariosExterno{}, err
	}
	b, err := json.Marshal(s.Catalogo)
	if err != nil {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	ch := sha256.Sum256(b)
	ref := func(i int) core.ReferenciaEntradaCatalogo {
		return core.ReferenciaEntradaCatalogo{CatalogoID: s.Catalogo.CatalogoID, CatalogoVersion: s.Catalogo.Version,
			CatalogoHuellaSHA256: hex.EncodeToString(ch[:]), EntradaClave: s.Catalogo.Entradas[i].Clave}
	}
	return ResumenMotivosUsuariosExterno{Estado: "preparado", HuellaPlan: h, Secuencia: s.SecuenciaEsperada + 1,
		EventoRef:      referenciaAltaContratacionTemporalDesarrollo("evento_", "usuarios-motivos\x00"+h),
		MotivoConsulta: ref(0), MotivoActualizacion: ref(1)}, nil
}

// PublicarMotivosUsuariosExterno usa el publicador V2 común, con su rol
// interno nominal y checkpoint. No publica perfiles ni otra familia.
func PublicarMotivosUsuariosExterno(ctx context.Context, con *pgx.Conn, s SolicitudMotivosUsuariosExterno, huella, aprobacion string) (r ResumenMotivosUsuariosExterno, fallo error) {
	r, err := s.Resumen()
	if err != nil || ctx == nil || ctx.Err() != nil || con == nil || huella != r.HuellaPlan || aprobacion != s.AprobacionRef {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	defer func() {
		if revertirProvisionExterna(ctx, tx) != nil {
			r, fallo = ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
		}
	}()
	if prepararCanalProvisionExterna(ctx, tx, "motivos") != nil {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	return publicarMotivosUsuariosEnTransaccion(ctx, tx, s, r)
}

type transaccionMotivosUsuariosExterno interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Commit(context.Context) error
}

func publicarMotivosUsuariosEnTransaccion(ctx context.Context, tx transaccionMotivosUsuariosExterno, s SolicitudMotivosUsuariosExterno, r ResumenMotivosUsuariosExterno) (ResumenMotivosUsuariosExterno, error) {
	contenido, err := contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo([]core.ReferenciaEntradaCatalogo{r.MotivoConsulta, r.MotivoActualizacion}, s.Catalogo.PublicadoEn)
	if err != nil {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	var publicada bool
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion.publicar_motivos_autorizacion_v2($1,$2,$3,$4,$5,$6,$7,$8::pg_catalog.jsonb)`,
		r.EventoRef, r.Secuencia, r.HuellaPlan, s.Catalogo.CatalogoID, s.Catalogo.Version, r.MotivoConsulta.CatalogoHuellaSHA256, s.Catalogo.PublicadoEn, contenido).Scan(&publicada)
	if err != nil || !publicada {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	if err := ctx.Err(); err != nil {
		return ResumenMotivosUsuariosExterno{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ResumenMotivosUsuariosExterno{}, errMotivosUsuariosExterno
	}
	r.Estado = "confirmado"
	return r, nil
}
