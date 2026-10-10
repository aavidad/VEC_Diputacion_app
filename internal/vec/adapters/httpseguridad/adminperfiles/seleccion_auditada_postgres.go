package adminperfiles

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

// plazoSelectorAuditado acota cada lectura o selección auditada, reintentos
// incluidos.
const plazoSelectorAuditado = 10 * time.Second

const listarPropiosAuditadoSQL = `SELECT resultado,auditoria_comun_ref FROM vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
const seleccionarPerfilAuditadoSQL = `SELECT resultado,auditoria_comun_ref FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`

type seleccionAuditadaPostgreSQL struct {
	base *PostgreSQL
}

// NuevaSeleccionAuditadaPostgreSQL acredita el LOGIN exclusivo de IS14.
// No abre las fuentes posperfil ni concede acceso a una consulta de negocio.
func NuevaSeleccionAuditadaPostgreSQL(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj) (FuenteSeleccionAuditadaADMIN, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || nulo(reloj) {
		return nil, api.ErrConfiguracionIncompleta
	}
	base := &PostgreSQL{pool: pool, reloj: reloj}
	if err := acreditarSeleccionAuditada(ctx, base); err != nil {
		return nil, err
	}
	return &seleccionAuditadaPostgreSQL{base: base}, nil
}

func acreditarSeleccionAuditada(ctx context.Context, base *PostgreSQL) error {
	err := base.transaccion(ctx, func(tx pgx.Tx) error {
		var acreditada bool
		var proceso string
		if err := tx.QueryRow(ctx, `SELECT acreditada,proceso FROM vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1()`).Scan(&acreditada, &proceso); err != nil || !acreditada || !textoCatalogo(proceso, 80) {
			return api.ErrConfiguracionIncompleta
		}
		return nil
	})
	if err != nil {
		return api.ErrConfiguracionIncompleta
	}
	return nil
}

type resultadoSelectorAuditado struct {
	Estado          string `json:"estado,omitempty"`
	MotivoRef       string `json:"motivo_ref,omitempty"`
	Revision        uint64 `json:"revision,omitempty"`
	PerfilActivoRef string `json:"perfil_activo_ref,omitempty"`
	Perfiles        []struct {
		PerfilRef      string `json:"perfil_ref"`
		RolVersionRef  string `json:"rol_version_ref"`
		ClaveI18N      string `json:"clave_i18n"`
		CategoriaADMIN string `json:"categoria_admin"`
	} `json:"perfiles,omitempty"`
	PerfilRef         string    `json:"perfil_ref,omitempty"`
	SeleccionRevision uint64    `json:"seleccion_revision,omitempty"`
	SeleccionadaEn    time.Time `json:"seleccionada_en,omitempty"`
}

func (s *seleccionAuditadaPostgreSQL) ListarPropiosAuditadosADMIN(ctx context.Context, o ObservacionADMIN) (LecturaPropiosAuditadaADMIN, error) {
	var propios PerfilesPropios
	_, ref, err := s.consultar(ctx, o, listarPropiosAuditadoSQL, nil, func(x resultadoSelectorAuditado) error {
		if x.Perfiles == nil || len(x.Perfiles) == 0 || len(x.Perfiles) > 16 || x.Revision > 1<<53-1 || x.PerfilRef != "" || x.SeleccionRevision != 0 || !x.SeleccionadaEn.IsZero() {
			return ConClaseSelector("selector_lista_invalida", api.ErrConfiguracionIncompleta)
		}
		propios = PerfilesPropios{Revision: x.Revision, PerfilActivoRef: x.PerfilActivoRef}
		for _, p := range x.Perfiles {
			propios.Perfiles = append(propios.Perfiles, PerfilPropio{p.PerfilRef, p.RolVersionRef, p.ClaveI18N, p.CategoriaADMIN})
		}
		if !propios.Validos() {
			return ConClaseSelector("selector_lista_invalida", api.ErrConfiguracionIncompleta)
		}
		return nil
	})
	if err != nil {
		return LecturaPropiosAuditadaADMIN{}, err
	}
	return LecturaPropiosAuditadaADMIN{Propios: propios, AuditoriaComunRef: ref}, nil
}

func (s *seleccionAuditadaPostgreSQL) SeleccionarPerfilAuditadoADMIN(ctx context.Context, o ObservacionADMIN, perfil string, revision uint64) (SeleccionAuditadaADMIN, error) {
	if !referencia(perfil, "prf_") || revision >= 1<<53-1 {
		return SeleccionAuditadaADMIN{}, api.ErrConflictoEstado
	}
	var seleccion SeleccionPerfil
	_, ref, err := s.consultar(ctx, o, seleccionarPerfilAuditadoSQL, []any{perfil, strconv.FormatUint(revision, 10)}, func(x resultadoSelectorAuditado) error {
		x.SeleccionadaEn = x.SeleccionadaEn.UTC()
		switch {
		case x.PerfilRef != perfil:
			return ConClaseSelector("selector_perfil_distinto", api.ErrConfiguracionIncompleta)
		case x.SeleccionRevision == 0 || x.SeleccionRevision < revision || x.SeleccionRevision > revision+1:
			return ConClaseSelector("selector_revision_fuera_de_rango", api.ErrConfiguracionIncompleta)
		case !instante(x.SeleccionadaEn):
			return ConClaseSelector("selector_instante", api.ErrConfiguracionIncompleta)
		case x.Perfiles != nil || x.PerfilActivoRef != "" || x.Revision != 0:
			return ConClaseSelector("selector_campos_extra", api.ErrConfiguracionIncompleta)
		case x.SeleccionadaEn.After(s.base.reloj.Ahora()):
			return ConClaseSelector("selector_seleccion_futura", api.ErrConfiguracionIncompleta)
		}
		seleccion = SeleccionPerfil{PerfilActivoRef: perfil, Revision: x.SeleccionRevision, SeleccionadaEn: x.SeleccionadaEn}
		return nil
	})
	if err != nil {
		return SeleccionAuditadaADMIN{}, err
	}
	seleccion.AuditoriaRef = ref
	return SeleccionAuditadaADMIN{Seleccion: seleccion, AuditoriaComunRef: ref}, nil
}

func (s *seleccionAuditadaPostgreSQL) consultar(ctx context.Context, o ObservacionADMIN, consulta string, extra []any, validar func(resultadoSelectorAuditado) error) (resultadoSelectorAuditado, string, error) {
	var salida resultadoSelectorAuditado
	if s == nil || s.base == nil || ctx == nil || ctx.Err() != nil || nulo(s.base.pool) || nulo(s.base.reloj) || !o.Valida(s.base.reloj.Ahora().UTC()) {
		return salida, "", api.ErrAutenticacionRequerida
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		return salida, "", ConClaseSelector("selector_correlacion", api.ErrConfiguracionIncompleta)
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return salida, "", ConClaseSelector("selector_aleatorio", api.ErrConfiguracionIncompleta)
	}
	hexEvento := hex.EncodeToString(nonce[:])
	args := append(argumentos(o), extra...)
	args = append(args, "evento_"+hexEvento, "correlacion_"+correlacion)
	var ref string
	var denegada error
	var resultadoValidado bool
	// Cada activo de la página pasa por este selector y escribe en la cadena
	// común de auditoría; con varias peticiones a la vez PostgreSQL aborta
	// algunas con 40001. El aborto revierte la transacción entera, así que se
	// repite con otra nueva según la política común. Nunca se repite una
	// respuesta ya validada: su COMMIT puede haberse aplicado.
	// vec-admin no acota el contexto de la petición: el plazo propio limita
	// los reintentos muy por debajo de los 45 s de lectura y escritura HTTP.
	ctx, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(plazoSelectorAuditado))
	defer cancelar()
	err := postgresqlcomun.RepetirTrasCarreraSerializable(ctx, func() error {
		salida, ref, denegada, resultadoValidado = resultadoSelectorAuditado{}, "", nil, false
		// La observación mTLS debe seguir vigente en cada intento.
		if !o.Valida(s.base.reloj.Ahora().UTC()) {
			return api.ErrAutenticacionRequerida
		}
		err := s.base.transaccion(ctx, func(tx pgx.Tx) error {
			var bruto []byte
			if err := tx.QueryRow(ctx, consulta, args...).Scan(&bruto, &ref); err != nil {
				return ConClaseSelector(claseSQLSelector(err), err)
			}
			if ref != "aud_v3_p_"+hexEvento {
				return ConClaseSelector("selector_auditoria_ref", api.ErrConfiguracionIncompleta)
			}
			if decodificarSelectorAuditado(bruto, &salida) != nil {
				return ConClaseSelector("selector_decodificacion", api.ErrConfiguracionIncompleta)
			}
			if salida.Estado == "denegado" {
				if salida.Perfiles != nil || salida.PerfilActivoRef != "" || salida.PerfilRef != "" || salida.Revision != 0 || salida.SeleccionRevision != 0 || !salida.SeleccionadaEn.IsZero() {
					return ConClaseSelector("selector_denegacion_incoherente", api.ErrConfiguracionIncompleta)
				}
				switch salida.MotivoRef {
				case "seleccion_revision_obsoleta":
					denegada = api.ErrConflictoEstado
				case "seleccion_material_invalido":
					denegada = api.ErrAccesoDenegado
				case "perfil_propio_no_acreditado":
					denegada = api.ErrAccesoDenegado
				default:
					return ConClaseSelector("selector_motivo_desconocido", api.ErrConfiguracionIncompleta)
				}
				// La denegación acreditada ya tiene registro común. Confirmamos
				// primero ese asiento; después devolvemos el error al transporte.
				resultadoValidado = true
				return nil
			}
			if salida.Estado != "" || salida.MotivoRef != "" {
				return ConClaseSelector("selector_estado_inesperado", api.ErrConfiguracionIncompleta)
			}
			if err := validar(salida); err != nil {
				return err
			}
			resultadoValidado = true
			return nil
		})
		// Un COMMIT abortado por serialización no aplicó nada y se repite;
		// cualquier otro fallo tras validar es incierto y no se repite.
		if err != nil && resultadoValidado && !postgresqlcomun.EsCarreraSerializable(err) {
			return ConClaseSelector("selector_commit_incierto", conservarClaseSelector(err, api.ErrConfiguracionIncompleta))
		}
		return err
	})
	if err != nil {
		if resultadoValidado {
			// El COMMIT puede ser incierto. No atribuimos una denegación
			// funcional al error de cerrar una respuesta ya validada. Si fue una
			// carrera de serialización (no aplicó nada), la clase lo dice.
			clase := "selector_commit_incierto"
			if postgresqlcomun.EsCarreraSerializable(err) {
				clase = "selector_carrera_agotada"
			}
			return resultadoSelectorAuditado{}, "", ConClaseSelector(clase,
				conservarClaseSelector(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible))
		}
		if postgresqlcomun.EsCarreraSerializable(err) {
			// Agotados los reintentos, la colisión técnica no acredita que la
			// revisión seleccionada por la persona esté obsoleta.
			return resultadoSelectorAuditado{}, "", ConClaseSelector("selector_carrera_agotada",
				ports.ErrAutoridadAdministracionPerfilesNoDisponible)
		}
		return resultadoSelectorAuditado{}, "", conservarClaseSelector(err, errorAutoridad(err))
	}
	if denegada != nil {
		return resultadoSelectorAuditado{}, "", denegada
	}
	return salida, ref, nil
}

func decodificarSelectorAuditado(bruto []byte, x *resultadoSelectorAuditado) error {
	if len(bruto) == 0 || len(bruto) > 32768 {
		return api.ErrConfiguracionIncompleta
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(x) != nil || d.Decode(new(any)) != io.EOF {
		return api.ErrConfiguracionIncompleta
	}
	return nil
}
