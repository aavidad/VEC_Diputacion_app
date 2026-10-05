package adminperfiles

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	is "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func NuevoPostgreSQLConFuenteADMIN(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj,
	fuente FuenteIdentificadoresADMIN, seudonimizador SeudonimizadorFuenteADMIN) (*PostgreSQL, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || nulo(reloj) || nulo(fuente) || nulo(seudonimizador) {
		return nil, api.ErrConfiguracionIncompleta
	}
	p := &PostgreSQL{pool: pool, reloj: reloj, identificadores: fuente, seudonimizador: seudonimizador}
	err := p.transaccion(ctx, func(tx pgx.Tx) error {
		var login, proceso string
		var acreditada bool
		if err := tx.QueryRow(ctx, `SELECT identidad_login,acreditada,proceso FROM vec_identidad_sesiones_v1.acreditar_runtime_admin_perfiles_v1()`).Scan(&login, &acreditada, &proceso); err != nil || !acreditada || login == "" || !textoCatalogo(proceso, 80) {
			return api.ErrConfiguracionIncompleta
		}
		return nil
	})
	if err != nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	return p, nil
}

// toleranciaRelojAcuseIS16 admite que el reloj de PostgreSQL vaya algo por
// delante del de la aplicación: un acuse fechado hasta dos segundos después
// de «ahora» no se trata como fabricado.
const toleranciaRelojAcuseIS16 = 2 * time.Second

type acuseIS16 struct {
	Referencia   string    `json:"auditoria_ref"`
	Secuencia    uint64    `json:"secuencia"`
	Huella       string    `json:"huella_sha256"`
	Correlacion  string    `json:"correlacion_ref"`
	RegistradaEn time.Time `json:"registrada_en"`
}

func leerResultadoIS16(bruto, acuse []byte, evento, correlacion string, ahora time.Time) (json.RawMessage, error, error) {
	var a acuseIS16
	if jsonCerradoIS16(acuse, &a) != nil || !hexConPrefijoIS16(evento, "evento_") || a.Referencia != "aud_v3_ap2_"+evento[7:] || a.Secuencia == 0 || a.Secuencia > 1<<53-1 ||
		!huella(a.Huella) || a.Correlacion != correlacion || !instante(a.RegistradaEn.UTC()) || a.RegistradaEn.After(ahora.Add(toleranciaRelojAcuseIS16)) {
		return nil, nil, api.ErrConfiguracionIncompleta
	}
	var x struct {
		Estado string          `json:"estado"`
		Datos  json.RawMessage `json:"datos"`
	}
	if jsonCerradoIS16(bruto, &x) != nil {
		return nil, nil, api.ErrConfiguracionIncompleta
	}
	if x.Estado == "denegado" && bytes.Equal(x.Datos, []byte("null")) {
		return nil, api.ErrAccesoDenegado, nil
	}
	if x.Estado == "error" && bytes.Equal(x.Datos, []byte("null")) {
		return nil, api.ErrConfiguracionIncompleta, nil
	}
	if x.Estado != "permitido" || len(x.Datos) == 0 || bytes.Equal(x.Datos, []byte("null")) {
		return nil, nil, api.ErrConfiguracionIncompleta
	}
	return x.Datos, nil, nil
}

func jsonCerradoIS16(bruto []byte, destino any) error {
	if len(bruto) == 0 || len(bruto) > 32768 {
		return api.ErrConfiguracionIncompleta
	}
	lexico := json.NewDecoder(bytes.NewReader(bruto))
	if valorUnicoIS16(lexico, 0) != nil {
		return api.ErrConfiguracionIncompleta
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return api.ErrConfiguracionIncompleta
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return api.ErrConfiguracionIncompleta
	}
	return nil
}

func valorUnicoIS16(d *json.Decoder, profundidad int) error {
	if profundidad > 8 {
		return api.ErrConfiguracionIncompleta
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, compuesta := t.(json.Delim)
	if !compuesta {
		return nil
	}
	if delim != '{' && delim != '[' {
		return api.ErrConfiguracionIncompleta
	}
	claves := make(map[string]bool)
	for d.More() {
		if delim == '{' {
			k, err := d.Token()
			if err != nil {
				return err
			}
			clave, ok := k.(string)
			if !ok || claves[clave] {
				return api.ErrConfiguracionIncompleta
			}
			claves[clave] = true
		}
		if err := valorUnicoIS16(d, profundidad+1); err != nil {
			return err
		}
	}
	cierre, err := d.Token()
	if err != nil || (delim == '{' && cierre != json.Delim('}')) || (delim == '[' && cierre != json.Delim(']')) {
		return api.ErrConfiguracionIncompleta
	}
	return nil
}

func hexConPrefijoIS16(s, prefijo string) bool {
	return len(s) == len(prefijo)+32 && s[:len(prefijo)] == prefijo && hexVinculo(s[len(prefijo):])
}

func argumentosAuditadosADMIN(ctx context.Context, o ObservacionADMIN) ([]any, error) {
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok || !hexVinculo(correlacion) {
		return nil, api.ErrConfiguracionIncompleta
	}
	var evento [16]byte
	if _, err := rand.Read(evento[:]); err != nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	return append(argumentos(o), "evento_"+hex.EncodeToString(evento[:]), "correlacion_"+correlacion), nil
}

type cuentaSQLIS16 struct {
	PersonaRef          string    `json:"persona_ref"`
	CuentaRef           string    `json:"cuenta_ref"`
	CuentaOrdinariaRef  string    `json:"cuenta_ordinaria_ref"`
	PerfilRef           string    `json:"perfil_activo_ref"`
	RolID               string    `json:"rol_id"`
	VinculoRef          string    `json:"vinculo_ref"`
	VinculoVersion      uint64    `json:"vinculo_version"`
	PoliticaRef         string    `json:"politica_garantia_ref"`
	PoliticaSHA256      string    `json:"politica_garantia_huella_sha256"`
	Garantia            string    `json:"garantia_observada"`
	VigenteHasta        time.Time `json:"vigente_hasta"`
	SeleccionRevision   uint64    `json:"seleccion_revision"`
	EspacioIdentidad    string    `json:"espacio_identidad"`
	EsquemaHMAC         string    `json:"esquema_hmac"`
	DominioHMAC         string    `json:"dominio_hmac_ref"`
	ClaveHMACID         string    `json:"clave_hmac_id"`
	ClaveHMACVersion    uint64    `json:"clave_hmac_version"`
	SujetoHMAC          string    `json:"sujeto_hmac_hex"`
	CuentaHMAC          string    `json:"cuenta_hmac_hex"`
	CuentaOrdinariaHMAC string    `json:"cuenta_ordinaria_hmac_hex"`
	FuenteRef           string    `json:"fuente_ref"`
	FuenteSHA256        string    `json:"fuente_sha256"`
}

func (p *PostgreSQL) cuentaDesdeFuente(ctx context.Context, o ObservacionADMIN, bruto []byte) (CuentaADMIN, error) {
	var x cuentaSQLIS16
	if jsonCerradoIS16(bruto, &x) != nil {
		return CuentaADMIN{}, api.ErrConfiguracionIncompleta
	}
	r := ReferenciaFuenteIdentificadoresADMIN{PersonaRef: x.PersonaRef, CuentaRef: x.CuentaRef, CuentaOrdinariaRef: x.CuentaOrdinariaRef,
		CertificadoSHA256: o.CertificadoSHA256, CASHA256: o.CASHA256, EspacioIdentidad: x.EspacioIdentidad, EsquemaHMAC: x.EsquemaHMAC,
		DominioHMACRef: x.DominioHMAC, ClaveHMACID: x.ClaveHMACID, ClaveHMACVersion: x.ClaveHMACVersion, FuenteRef: x.FuenteRef, FuenteSHA256: x.FuenteSHA256}
	for _, par := range []struct {
		texto   string
		destino *[32]byte
	}{{x.SujetoHMAC, &r.SujetoHMAC}, {x.CuentaHMAC, &r.CuentaHMAC}, {x.CuentaOrdinariaHMAC, &r.CuentaOrdinariaHMAC}} {
		if !huella(par.texto) {
			return CuentaADMIN{}, api.ErrConfiguracionIncompleta
		}
		b, err := hex.DecodeString(par.texto)
		if err != nil {
			return CuentaADMIN{}, api.ErrConfiguracionIncompleta
		}
		copy(par.destino[:], b)
	}
	ids, err := cotejarIdentificadoresFuenteADMIN(ctx, p.identificadores, p.seudonimizador, r)
	if err != nil {
		return CuentaADMIN{}, err
	}
	c := CuentaADMIN{SujetoID: ids.SujetoID, CuentaID: ids.CuentaID, CuentaOrdinariaID: ids.CuentaOrdinariaID,
		PersonaRef: x.PersonaRef, CuentaRef: x.CuentaRef, CuentaOrdinariaRef: x.CuentaOrdinariaRef, PerfilActivoRef: x.PerfilRef,
		RolID: x.RolID, VinculoRef: x.VinculoRef, VinculoVersion: x.VinculoVersion, SeleccionRevision: x.SeleccionRevision,
		PoliticaGarantiaRef: x.PoliticaRef, PoliticaGarantiaHuellaSHA256: x.PoliticaSHA256, GarantiaObservada: domain.AuthAssurance(x.Garantia), VigenteHasta: x.VigenteHasta.UTC()}
	if !c.Valida(p.reloj.Ahora().UTC()) {
		return CuentaADMIN{}, api.ErrAccesoDenegado
	}
	var material map[string]json.RawMessage
	if json.Unmarshal(bruto, &material) != nil {
		return CuentaADMIN{}, api.ErrConfiguracionIncompleta
	}
	delete(material, "vigente_hasta")
	b, err := json.Marshal(material)
	if err != nil {
		return CuentaADMIN{}, api.ErrConfiguracionIncompleta
	}
	c.materialCuentaSQL = string(b)
	return c, nil
}

func cotejarIdentificadoresFuenteADMIN(ctx context.Context, fuente FuenteIdentificadoresADMIN, seud SeudonimizadorFuenteADMIN, r ReferenciaFuenteIdentificadoresADMIN) (IdentificadoresFuenteADMIN, error) {
	if ctx == nil || ctx.Err() != nil || nulo(fuente) || nulo(seud) || r.EsquemaHMAC != is.EsquemaHMACSHA256V1 || !huella(r.FuenteSHA256) ||
		!referencia(r.PersonaRef, "per_") || !referencia(r.CuentaRef, "cta_") || !referencia(r.CuentaOrdinariaRef, "cta_") || r.CuentaRef == r.CuentaOrdinariaRef ||
		!huella(r.CertificadoSHA256) || !huella(r.CASHA256) || r.ClaveHMACVersion == 0 || r.ClaveHMACVersion > 1<<63-1 ||
		r.SujetoHMAC == [32]byte{} || r.CuentaHMAC == [32]byte{} || r.CuentaOrdinariaHMAC == [32]byte{} {
		return IdentificadoresFuenteADMIN{}, api.ErrConfiguracionIncompleta
	}
	ids, err := fuente.ResolverIdentificadoresADMIN(ctx, r)
	if err != nil || ids.EspacioIdentidad != r.EspacioIdentidad || ids.DominioHMACRef != r.DominioHMACRef || ids.ClaveHMACID != r.ClaveHMACID ||
		ids.ClaveHMACVersion != r.ClaveHMACVersion || ids.FuenteRef != r.FuenteRef || ids.FuenteSHA256 != r.FuenteSHA256 ||
		!identificadorOriginalADMIN(ids.SujetoID) || !identificadorOriginalADMIN(ids.CuentaID) || !identificadorOriginalADMIN(ids.CuentaOrdinariaID) {
		return IdentificadoresFuenteADMIN{}, api.ErrConfiguracionIncompleta
	}
	// SeudonimizarAlta también requiere referencias aleatorias de aserción y
	// sesión. Estas dos se descartan; las tres preimágenes de identidad son
	// exclusivamente las entregadas por el productor original.
	var aleatorio [32]byte
	if _, err = rand.Read(aleatorio[:]); err != nil {
		return IdentificadoresFuenteADMIN{}, api.ErrConfiguracionIncompleta
	}
	d, aliasOrdinario, err := is.SeudonimizarAltaConAliasCuentaOrdinaria(ctx, seud, is.IdentificadoresAlta{EspacioIdentidad: r.EspacioIdentidad, AsercionID: hex.EncodeToString(aleatorio[:16]), SesionID: hex.EncodeToString(aleatorio[16:]), SujetoID: ids.SujetoID, CuentaID: ids.CuentaID, CuentaOrdinariaID: ids.CuentaOrdinariaID}, r.EspacioIdentidad, r.DominioHMACRef)
	if err != nil || d.Esquema != r.EsquemaHMAC || d.EspacioIdentidad != r.EspacioIdentidad || d.DominioRef != r.DominioHMACRef || d.ClaveID != r.ClaveHMACID || d.ClaveVersion != r.ClaveHMACVersion ||
		subtle.ConstantTimeCompare(d.SujetoIDHMAC[:], r.SujetoHMAC[:]) != 1 || subtle.ConstantTimeCompare(d.CuentaIDHMAC[:], r.CuentaHMAC[:]) != 1 ||
		subtle.ConstantTimeCompare(aliasOrdinario, r.CuentaOrdinariaHMAC[:]) != 1 {
		return IdentificadoresFuenteADMIN{}, api.ErrConfiguracionIncompleta
	}
	return ids, nil
}

func (p *PostgreSQL) VincularSesionADMINConAcuse(ctx context.Context, o ObservacionADMIN, c CuentaADMIN, refs ReferenciasSesionADMIN) (VinculoSesionADMIN, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.pool) || nulo(p.reloj) || !o.Valida(p.reloj.Ahora().UTC()) || !c.Valida(p.reloj.Ahora().UTC()) ||
		c.materialCuentaSQL == "" || !referencia(refs.AutenticacionRef, "aut_") || !referencia(refs.SesionRef, "ses_") {
		return VinculoSesionADMIN{}, api.ErrAutenticacionRequerida
	}
	args, err := argumentosAuditadosADMIN(ctx, o)
	if err != nil {
		return VinculoSesionADMIN{}, err
	}
	var entropia [16]byte
	if _, err = rand.Read(entropia[:]); err != nil {
		return VinculoSesionADMIN{}, api.ErrConfiguracionIncompleta
	}
	vis := "vis_" + hex.EncodeToString(entropia[:])
	evento, correlacion := args[9], args[10]
	args = append(args[:9], refs.AutenticacionRef, refs.SesionRef, json.RawMessage(c.materialCuentaSQL), vis, evento, correlacion)
	var v VinculoSesionADMIN
	var decision error
	err = p.transaccion(ctx, func(tx pgx.Tx) error {
		sub, err := tx.Begin(ctx)
		if err != nil {
			return api.ErrConfiguracionIncompleta
		}
		defer sub.Rollback(ctx)
		var bruto, acuse []byte
		if err := sub.QueryRow(ctx, vincularSesion, args...).Scan(&bruto, &acuse); err != nil {
			return errorConsultaIS16(err)
		}
		datos, denegada, err := leerResultadoIS16(bruto, acuse, evento.(string), correlacion.(string), p.reloj.Ahora())
		if err != nil {
			return err
		}
		decision = denegada
		// La decisión funcional se devuelve después de confirmar su auditoría.
		if decision != nil {
			return confirmarSubtransaccionIS16(ctx, sub)
		}
		v, err = p.cotejarVinculoIS16(datos, vis, o, c, refs)
		if err == nil {
			return confirmarSubtransaccionIS16(ctx, sub)
		}
		// SQL dio el vínculo por bueno pero Go no lo reconoce: se deshace el
		// vínculo y su acuse favorable y queda auditado un error en su lugar.
		v = VinculoSesionADMIN{}
		if err = sub.Rollback(ctx); err != nil {
			return api.ErrConfiguracionIncompleta
		}
		decision, err = p.rechazarCotejoIS16(ctx, tx, evento.(string), correlacion.(string), "vincular_sesion_admin")
		return err
	})
	if err != nil {
		return VinculoSesionADMIN{}, errorAutoridad(err)
	}
	if decision != nil {
		return VinculoSesionADMIN{}, decision
	}
	return v, nil
}

// cotejarVinculoIS16 comprueba que el vínculo devuelto por SQL es exactamente
// el pedido para esta cuenta, observación y sesión, y que sigue vigente.
func (p *PostgreSQL) cotejarVinculoIS16(datos []byte, vis string, o ObservacionADMIN, c CuentaADMIN, refs ReferenciasSesionADMIN) (VinculoSesionADMIN, error) {
	v, err := decodificarVinculoIS16(datos)
	if err != nil {
		return VinculoSesionADMIN{}, err
	}
	if v.Referencia != vis || v.AutenticacionRef != refs.AutenticacionRef || v.SesionRef != refs.SesionRef ||
		v.PersonaRef != c.PersonaRef || v.CuentaRef != c.CuentaRef || v.CuentaOrdinariaRef != c.CuentaOrdinariaRef || v.PerfilActivoRef != c.PerfilActivoRef ||
		v.SeleccionRevision != c.SeleccionRevision || v.CertificadoSHA256 != o.CertificadoSHA256 || v.CASHA256 != o.CASHA256 ||
		v.VinculoCertificadoRef != c.VinculoRef || v.VinculoCertificadoVersion != c.VinculoVersion || v.PoliticaRef != c.PoliticaGarantiaRef || v.PoliticaSHA256 != c.PoliticaGarantiaHuellaSHA256 ||
		!p.reloj.Ahora().Before(v.VigenteHasta) {
		return VinculoSesionADMIN{}, api.ErrConfiguracionIncompleta
	}
	return v, nil
}

const rechazarCotejo = `SELECT resultado,acuse FROM vec_identidad_sesiones_v1.rechazar_fuente_cuenta_admin_v1($1,$2,$3)`

// rechazarCotejoIS16 audita como error un resultado favorable de SQL que Go no
// ha podido cotejar. Se ejecuta fuera del SAVEPOINT ya revertido; sólo un
// envelope «error» con acuse válido cuenta como auditoría confirmada.
func (p *PostgreSQL) rechazarCotejoIS16(ctx context.Context, tx pgx.Tx, evento, correlacion, accion string) (error, error) {
	var bruto, acuse []byte
	if err := tx.QueryRow(ctx, rechazarCotejo, evento, correlacion, accion).Scan(&bruto, &acuse); err != nil {
		return nil, errorConsultaIS16(err)
	}
	_, decision, err := leerResultadoIS16(bruto, acuse, evento, correlacion, p.reloj.Ahora())
	if err != nil || decision == nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	return decision, nil
}

type vinculoSQLIS16 struct {
	Referencia                string    `json:"referencia"`
	Version                   uint64    `json:"version"`
	Huella                    string    `json:"huella_sha256"`
	Autenticacion             string    `json:"autenticacion_ref"`
	Sesion                    string    `json:"sesion_ref"`
	Persona                   string    `json:"persona_ref"`
	Cuenta                    string    `json:"cuenta_ref"`
	Ordinaria                 string    `json:"cuenta_ordinaria_ref"`
	Perfil                    string    `json:"perfil_activo_ref"`
	Certificado               string    `json:"certificado_sha256"`
	CA                        string    `json:"ca_sha256"`
	VinculoCertificado        string    `json:"vinculo_certificado_ref"`
	VinculoCertificadoVersion uint64    `json:"vinculo_certificado_version"`
	Politica                  string    `json:"politica_ref"`
	PoliticaSHA               string    `json:"politica_sha256"`
	Seleccion                 uint64    `json:"seleccion_revision"`
	Control                   string    `json:"control_sesion_ref"`
	ControlRevision           uint64    `json:"control_sesion_revision"`
	ControlSHA                string    `json:"control_sesion_sha256"`
	Vinculada                 time.Time `json:"vinculada_en"`
	Hasta                     time.Time `json:"vigente_hasta"`
	Fuente                    string    `json:"fuente_ref"`
	FuenteSHA                 string    `json:"fuente_sha256"`
}

func decodificarVinculoIS16(bruto []byte) (VinculoSesionADMIN, error) {
	var x vinculoSQLIS16
	if jsonCerradoIS16(bruto, &x) != nil {
		return VinculoSesionADMIN{}, api.ErrConfiguracionIncompleta
	}
	v := VinculoSesionADMIN{Referencia: x.Referencia, Version: x.Version, HuellaSHA256: x.Huella, AutenticacionRef: x.Autenticacion, SesionRef: x.Sesion, PersonaRef: x.Persona,
		CuentaRef: x.Cuenta, CuentaOrdinariaRef: x.Ordinaria, PerfilActivoRef: x.Perfil, CertificadoSHA256: x.Certificado, CASHA256: x.CA, VinculoCertificadoRef: x.VinculoCertificado,
		VinculoCertificadoVersion: x.VinculoCertificadoVersion, PoliticaRef: x.Politica, PoliticaSHA256: x.PoliticaSHA, SeleccionRevision: x.Seleccion,
		ControlSesionRef: x.Control, ControlSesionRevision: x.ControlRevision, ControlSesionSHA256: x.ControlSHA, VinculadaEn: x.Vinculada.UTC(), VigenteHasta: x.Hasta.UTC(), FuenteRef: x.Fuente, FuenteSHA256: x.FuenteSHA}
	if v.Validar() != nil || v.FuenteRef != "vinculo_sesion_admin:"+v.Referencia[4:] || v.FuenteSHA256 != v.HuellaSHA256 {
		return VinculoSesionADMIN{}, api.ErrConfiguracionIncompleta
	}
	return v, nil
}

var _ FuenteCuentasADMINConAcuse = (*PostgreSQL)(nil)

// Una denegación funcional sólo procede del envelope confirmado con ACK.
// Un 42501 técnico no tiene ese acuse; no se presenta como decisión funcional.
func errorConsultaIS16(err error) error {
	var pg *pgconn.PgError
	// Serialización e interbloqueo son conflictos reintentables, no fallos de
	// configuración.
	if errors.As(err, &pg) && (pg.Code == "40001" || pg.Code == "40P01") {
		return api.ErrConflictoEstado
	}
	return api.ErrConfiguracionIncompleta
}

func confirmarSubtransaccionIS16(ctx context.Context, tx pgx.Tx) error {
	if tx.Commit(ctx) != nil {
		return api.ErrConfiguracionIncompleta
	}
	return nil
}
