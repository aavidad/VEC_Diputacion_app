package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/auditoria"
	"vec-diputacion-granada/internal/vec/ports"
)

// Nombres fijos dentro del directorio privado de salida. Ningún fichero se
// sobrescribe: un segundo preparar en la misma carpeta se rechaza.
const (
	ArchivoConfiguracionGobiernoUsuarios = "configuracion-material.json"
	ArchivoMaterialGobiernoUsuarios      = "material.json"
	ArchivoPlanGobiernoUsuarios          = "plan.json"
	ArchivoAprobacionGobiernoUsuarios    = "aprobacion-candidata.json"
)

// Ambos errores siguen siendo ErrGobiernoUsuariosAdmin para errors.Is, pero
// distinguen los dos casos en que el resultado no es un fallo limpio:
// COMMIT enviado sin confirmación y COMMIT confirmado sin acuse guardado.
var (
	ErrCommitGobiernoUsuariosIndeterminado = fmt.Errorf("%w: commit_indeterminado", ErrGobiernoUsuariosAdmin)
	ErrAcuseGobiernoUsuariosNoGuardado     = fmt.Errorf("%w: acuse_no_guardado", ErrGobiernoUsuariosAdmin)
)

const limiteArchivoGobiernoUsuarios = 16384

// MaterialOrigenGobiernoUsuariosAdmin señala el material privado existente:
// proveedor HMAC y semilla del firmante de la raíz ya publicada.
type MaterialOrigenGobiernoUsuariosAdmin struct {
	DirectorioMaterial, RutaConfiguracionHMAC, ArchivoSemillaRaiz string
	// ValidezClaves acota la ventana de las claves nuevas (1 h a 24 h).
	ValidezClaves time.Duration
	// ConjuntoVersion 0: gobierno de usuarios (AD188). 1 o más: conjunto de
	// capacidades ADMIN de AD198 (incluye el lote ordinario de perfiles).
	ConjuntoVersion uint64
}

// PreparacionGobiernoUsuariosAdmin devuelve sólo huellas: el DBA las fija en
// config_gobierno_usuarios_admin_v1 antes de aplicar.
type PreparacionGobiernoUsuariosAdmin struct {
	PlanSHA256      string    `json:"plan_sha256"`
	MaterialSHA256  string    `json:"material_sha256"`
	PreimagenSHA256 string    `json:"preimagen_sha256"`
	CaducaEn        time.Time `json:"caduca_en"`
}

type instantaneaGobiernoUsuarios struct {
	Revision    string    `json:"revision"`
	Secuencia   uint64    `json:"secuencia"`
	SPKI        string    `json:"spki"`
	ClaveID     string    `json:"clave_id"`
	Version     uint64    `json:"version"`
	Audiencia   string    `json:"audiencia"`
	Desde       time.Time `json:"desde"`
	Hasta       time.Time `json:"hasta"`
	Orden       uint64    `json:"orden"`
	MaxVersion  uint64    `json:"max_version"`
	MaxRevision uint64    `json:"max_revision"`
	PreSHA      string    `json:"pre_sha"`
}

const consultaInstantaneaGobiernoUsuarios = `SELECT jsonb_build_object('revision',c.revision,'secuencia',c.secuencia,'spki',encode(r.clave_publica_spki,'base64'),'clave_id',r.clave_id,'version',r.version,'audiencia',r.audiencia_despliegue,'desde',r.valida_desde,'hasta',r.valida_hasta,'orden',(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),'max_version',(SELECT max(version) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'max_revision',(SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'pre_sha',encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()::text,'UTF8')),'hex')) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON r.clave_id=cr.raiz_clave_id AND r.version=cr.raiz_version ORDER BY p.orden DESC LIMIT 1`

// descriptoresClavesUsuariosAdmin fija las dos claves de capacidad de una
// publicación. La derivación es determinista y AD188 rechaza un clave_id o un
// secreto ya publicados; con dominio y prefijo fijos sólo podía publicarse una
// vez. La secuencia del gobierno, única y creciente, entra en el dominio de
// derivación y en el identificador: cada renovación obtiene claves nuevas y
// repetir la preparación de una publicación no aplicada da las mismas.
// Una clave por audiencia del conjunto, en su orden. Para el conjunto 0
// produce exactamente los descriptores de AD188 (usuarios listar/consultar).
func descriptoresClavesUsuariosAdmin(conjunto []AudienciaCapacidadAdmin, secuencia, maxVersion, maxRevision uint64, ahora time.Time, validez time.Duration) []DescriptorClaveUsuariosAdmin {
	s := strconv.FormatUint(secuencia, 10)
	var entradas []DescriptorClaveUsuariosAdmin
	for i, e := range conjunto {
		entradas = append(entradas, DescriptorClaveUsuariosAdmin{Audiencia: e.Audiencia, Dominio: "vec.admin.desarrollo." + strings.ReplaceAll(e.Segmento, ":", ".") + ".capacidad-v3.s" + s, PrefijoClave: "clave:capacidad:admin:" + e.Segmento + ":s" + s + ":", EmisorID: e.EmisorID, Version: maxVersion + uint64(i) + 1, RevisionGobierno: maxRevision + uint64(i) + 1, ValidaDesde: ahora.Add(-time.Minute), ValidaHasta: ahora.Add(validez)})
	}
	return entradas
}

// La misma instantánea con la preimagen del conjunto de AD198 ($1).
var consultaInstantaneaGobiernoCapacidades = strings.Replace(consultaInstantaneaGobiernoUsuarios,
	"vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()",
	"vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1($1::integer)", 1)

// PrepararGobiernoUsuariosAdmin lee la configuración vigente con un pool de
// lectura, deriva las dos claves con el proveedor existente y escribe plan,
// material y configuración en la carpeta privada. No publica nada.
func PrepararGobiernoUsuariosAdmin(ctx context.Context, lectura *pgxpool.Pool, origen MaterialOrigenGobiernoUsuariosAdmin, salida *os.Root, reloj ports.Reloj) (PreparacionGobiernoUsuariosAdmin, error) {
	var vacio PreparacionGobiernoUsuariosAdmin
	if ctx == nil || lectura == nil || salida == nil || dependenciaBootstrapNula(reloj) || origen.ValidezClaves < time.Hour || origen.ValidezClaves > 24*time.Hour {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	conjunto, ok := AudienciasConjuntoCapacidadesAdmin(origen.ConjuntoVersion)
	if !ok {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	consulta, argumentos := consultaInstantaneaGobiernoUsuarios, []any{}
	if origen.ConjuntoVersion != 0 {
		consulta, argumentos = consultaInstantaneaGobiernoCapacidades, []any{int64(origen.ConjuntoVersion)}
	}
	var raw []byte
	if lectura.QueryRow(ctx, consulta, argumentos...).Scan(&raw) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var actual instantaneaGobiernoUsuarios
	if decodificarGobiernoUsuarios(raw, &actual) != nil || !shaGobiernoUsuarios.MatchString(actual.PreSHA) {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	pub, err := publicaRaizGobiernoUsuarios(actual.SPKI)
	if err != nil {
		return vacio, err
	}
	now := reloj.Ahora().UTC().Truncate(time.Microsecond)
	dia := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	cfg := ConfiguracionMaterialUsuariosAdmin{DirectorioMaterial: origen.DirectorioMaterial, RutaConfiguracionHMAC: origen.RutaConfiguracionHMAC, ArchivoSemillaRaiz: origen.ArchivoSemillaRaiz, PrefijoEvidencia: "evidencia:firma:admin:usuarios:", Raiz: administracion.MaterialRaizPerfilesV3{ClaveID: actual.ClaveID, Version: actual.Version, Audiencia: actual.Audiencia, Publica: pub, Estado: confianza.EstadoClaveAtestacionAutorizacionV3Activa, ValidaDesde: actual.Desde.UTC(), ValidaHasta: actual.Hasta.UTC()}}
	cfg.Gobierno = administracion.GobiernoConfianzaPerfilesV3{Revision: "confianza:atestacion:ct:desarrollo:" + dia.Format("2006-01-02") + ":r" + strconv.FormatUint(actual.Secuencia+1, 10), Secuencia: actual.Secuencia + 1, PublicadaEn: dia, ExpiraEn: dia.Add(24 * time.Hour)}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(cfg.Raiz.ClaveID, cfg.Raiz.Version, pub, cfg.Raiz.Audiencia, cfg.Raiz.Estado, cfg.Raiz.ValidaDesde, cfg.Raiz.ValidaHasta, time.Time{})
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	g, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(cfg.Gobierno.Revision, cfg.Gobierno.Secuencia, cfg.Gobierno.PublicadaEn, cfg.Gobierno.ExpiraEn, raiz)
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	if cfg.Gobierno.HuellaSHA256, err = g.HuellaSHA256ParaGobierno(); err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	cfg.ConjuntoVersion = origen.ConjuntoVersion
	cfg.Entradas = descriptoresClavesUsuariosAdmin(conjunto, cfg.Gobierno.Secuencia, actual.MaxVersion, actual.MaxRevision, now, origen.ValidezClaves)
	m, err := PrepararMaterialUsuariosAdmin(ctx, cfg, reloj)
	if err != nil {
		return vacio, err
	}
	defer m.Cerrar()
	var secreto bytes.Buffer
	defer func() { borrarBytes(secreto.Bytes()); secreto.Reset() }()
	if m.EscribirMaterialPrivado(&secreto) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	// El plan caduca a las dos horas como máximo y nunca después de las claves
	// ni de la configuración diaria, para no aplicar material ya vencido.
	caduca := now.Add(min(2*time.Hour, origen.ValidezClaves))
	if cfg.Gobierno.ExpiraEn.Before(caduca) {
		caduca = cfg.Gobierno.ExpiraEn
	}
	p := planGobiernoUsuariosAdmin{Version: 1, OperacionRef: "gcu_" + actual.PreSHA[:32], PreparadoEn: now, CaducaEn: caduca, PreimagenSHA256: actual.PreSHA}
	if origen.ConjuntoVersion != 0 {
		p.Version, p.OperacionRef, p.ConjuntoVersion = 2, "gca_"+actual.PreSHA[:32], origen.ConjuntoVersion
	}
	for i := range conjunto {
		p.Ordenes = append(p.Ordenes, actual.Orden+uint64(i)+1)
	}
	p.Configuracion.Revision = cfg.Gobierno.Revision
	p.Configuracion.Secuencia = cfg.Gobierno.Secuencia
	p.Configuracion.Huella = cfg.Gobierno.HuellaSHA256
	p.Configuracion.PublicadaEn = cfg.Gobierno.PublicadaEn
	p.Configuracion.ExpiraEn = cfg.Gobierno.ExpiraEn
	plan, err := json.Marshal(p)
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	datos, err := json.Marshal(cfg)
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	defer borrarBytes(datos)
	ph, mh := sha256.Sum256(plan), sha256.Sum256(secreto.Bytes())
	r := PreparacionGobiernoUsuariosAdmin{PlanSHA256: hex.EncodeToString(ph[:]), MaterialSHA256: hex.EncodeToString(mh[:]), PreimagenSHA256: actual.PreSHA, CaducaEn: p.CaducaEn}
	aprobacion, err := json.Marshal(r)
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	// La aprobación candidata se escribe la última: si falta, la preparación
	// quedó incompleta y la carpeta no debe usarse (no se sobrescribe nada).
	for _, f := range []struct {
		nombre string
		datos  []byte
	}{{ArchivoMaterialGobiernoUsuarios, secreto.Bytes()}, {ArchivoConfiguracionGobiernoUsuarios, datos}, {ArchivoPlanGobiernoUsuarios, plan}, {ArchivoAprobacionGobiernoUsuarios, aprobacion}} {
		if escribirPrivadoGobiernoUsuarios(salida, f.nombre, f.datos) != nil {
			return vacio, ErrGobiernoUsuariosAdmin
		}
	}
	return r, nil
}

// AplicarGobiernoUsuariosAdminPreparado reconstruye el material desde la
// configuración original y exige que coincida byte a byte con el preparado.
// Un replay usa el mismo plan y otro nombre de acuse.
func AplicarGobiernoUsuariosAdminPreparado(ctx context.Context, operador *pgxpool.Pool, salida *os.Root, nombreAcuse string, reloj ports.Reloj) (ConfirmacionGobiernoUsuariosAdmin, error) {
	var vacio ConfirmacionGobiernoUsuariosAdmin
	if ctx == nil || operador == nil || salida == nil || dependenciaBootstrapNula(reloj) || !nombreAcuseGobiernoUsuariosValido(nombreAcuse) {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	// El acuse se reserva antes de enviar nada: un destino existente se
	// rechaza sin tocar la base y sólo se conserva si llega a guardarse.
	archivo, err := reservarPrivadoGobiernoUsuarios(salida, nombreAcuse)
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	guardado := false
	defer func() {
		_ = archivo.Close()
		if !guardado {
			_ = salida.Remove(nombreAcuse)
		}
	}()
	data, err := leerPrivadoGobiernoUsuarios(salida, ArchivoConfiguracionGobiernoUsuarios)
	defer borrarBytes(data)
	var cfg ConfiguracionMaterialUsuariosAdmin
	if err != nil || decodificarGobiernoUsuarios(data, &cfg) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	// Las fechas proceden de los bytes originales, no del reloj.
	cfg.Raiz.ValidaDesde, cfg.Raiz.ValidaHasta = cfg.Raiz.ValidaDesde.UTC(), cfg.Raiz.ValidaHasta.UTC()
	cfg.Gobierno.PublicadaEn, cfg.Gobierno.ExpiraEn = cfg.Gobierno.PublicadaEn.UTC(), cfg.Gobierno.ExpiraEn.UTC()
	for i := range cfg.Entradas {
		cfg.Entradas[i].ValidaDesde, cfg.Entradas[i].ValidaHasta = cfg.Entradas[i].ValidaDesde.UTC(), cfg.Entradas[i].ValidaHasta.UTC()
	}
	m, err := PrepararMaterialUsuariosAdmin(ctx, cfg, reloj)
	if err != nil {
		return vacio, err
	}
	defer m.Cerrar()
	var secreto bytes.Buffer
	defer func() { borrarBytes(secreto.Bytes()); secreto.Reset() }()
	if m.EscribirMaterialPrivado(&secreto) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	original, err := leerPrivadoGobiernoUsuarios(salida, ArchivoMaterialGobiernoUsuarios)
	defer borrarBytes(original)
	if err != nil || !bytes.Equal(original, secreto.Bytes()) {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	plan, err := leerPrivadoGobiernoUsuarios(salida, ArchivoPlanGobiernoUsuarios)
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	h := sha256.Sum256(plan)
	acuse, err := AplicarGobiernoUsuariosAdmin(ctx, operador, string(plan), hex.EncodeToString(h[:]), m)
	if err != nil {
		return vacio, err
	}
	b, err := json.Marshal(acuse)
	if err != nil || completarPrivadoGobiernoUsuarios(archivo, b) != nil {
		// COMMIT ya confirmado: el llamante debe recuperar con otro acuse.
		return acuse, ErrAcuseGobiernoUsuariosNoGuardado
	}
	guardado = true
	return acuse, nil
}

// VerificarCadenaGobiernoUsuariosAdmin exporta el tramo técnico contiguo
// posterior al último evento de otra familia y lo recalcula. No atribuye
// cobertura global ni autenticidad externa del checkpoint.
func VerificarCadenaGobiernoUsuariosAdmin(ctx context.Context, lectura *pgxpool.Pool, salida *os.Root, nombre string) (auditoria.InformeVerificacion, error) {
	var vacio auditoria.InformeVerificacion
	if ctx == nil || lectura == nil || salida == nil || !nombreAcuseGobiernoUsuariosValido(nombre) {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var data []byte
	if lectura.QueryRow(ctx, consultaCadenaGobiernoUsuarios, auditoria.EsquemaVerificacionGobiernoUsuarios).Scan(&data) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var doc auditoria.DocumentoVerificacionMixta
	if decodificarGobiernoUsuarios(data, &doc) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	informe := auditoria.VerificarCadenaGobiernoUsuariosV1(doc, doc.Manifiesto, 100)
	b, err := json.Marshal(informe)
	if err != nil || escribirPrivadoGobiernoUsuarios(salida, nombre, b) != nil {
		return informe, ErrGobiernoUsuariosAdmin
	}
	return informe, nil
}

// Posiciones de la cadena: hasta el corte de AD207 coinciden con el número
// del asiento; después las da el eslabón y solo cuentan los asientos sellados.
const consultaCadenaGobiernoUsuarios = `WITH corte AS (SELECT secuencia n FROM vec_autorizacion_atestada_v3.control_cadena_auditoria),
cadena AS (
 SELECT a.*,a.secuencia posicion,NULL::jsonb eslabon,a.anterior_sha256 enlace_previo,a.huella_sha256 enlace
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a,corte WHERE a.secuencia<=corte.n
 UNION ALL
 SELECT a.*,e.posicion,jsonb_build_object('posicion',e.posicion,'secuencia',e.secuencia,'anterior_sha256',e.anterior_sha256,'eslabon_sha256',e.eslabon_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'sellado_en',to_char(e.sellado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),e.anterior_sha256,e.eslabon_sha256
 FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING (secuencia)
), rows AS (
 SELECT * FROM cadena
 WHERE tipo_registro IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')
 AND posicion > COALESCE((SELECT max(posicion) FROM cadena
 WHERE tipo_registro NOT IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')),0)
), rango AS(SELECT min(posicion) primero,max(posicion) ultimo,count(*) cuenta FROM rows)
 SELECT jsonb_build_object('esquema',$1::text,'manifiesto',jsonb_build_object('cadena_id','cadena:comun:interna','primera_secuencia',ra.primero,'ultima_secuencia',ra.ultimo,'registros',ra.cuenta,'anterior_sha256',(SELECT enlace_previo FROM rows ORDER BY posicion LIMIT 1),'cabeza_sha256',(SELECT enlace FROM rows ORDER BY posicion DESC LIMIT 1)),
 'registros',(SELECT jsonb_agg(jsonb_build_object('tipo_registro',a.tipo_registro) || CASE WHEN a.eslabon IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('eslabon',a.eslabon) END ||
 CASE WHEN a.tipo_registro='gobierno_usuarios_admin' THEN jsonb_build_object('gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('plan_sha256',a.plan_sha256,'preimagen_sha256',a.gobierno_usuarios_detalle->>'preimagen_sha256','configuracion_origen_ref',a.gobierno_usuarios_detalle->>'configuracion_origen_ref','configuracion_destino_ref',a.gobierno_usuarios_detalle->>'configuracion_destino_ref','claves_sha256',a.gobierno_usuarios_detalle->>'claves_sha256'))
 ELSE jsonb_build_object('intento_gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('solicitud_sha256',a.gobierno_usuarios_solicitud_sha256)) END
 ORDER BY a.posicion) FROM rows a)) FROM rango ra`

func publicaRaizGobiernoUsuarios(spki string) (ed25519.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(spki)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, ErrGobiernoUsuariosAdmin
	}
	return pub, nil
}

// nombreAcuseGobiernoUsuariosValido admite sólo un nombre base nuevo que no
// coincida con los ficheros que deja preparar ni con los del overlay B1.
func nombreAcuseGobiernoUsuariosValido(nombre string) bool {
	if nombre == "" || len(nombre) > 128 || nombre == "." || nombre == ".." || filepath.Base(nombre) != nombre || filepath.Clean(nombre) != nombre {
		return false
	}
	switch nombre {
	case ArchivoConfiguracionGobiernoUsuarios, ArchivoMaterialGobiernoUsuarios, ArchivoPlanGobiernoUsuarios, ArchivoAprobacionGobiernoUsuarios,
		ArchivoOverlayVersionBolsa, ArchivoMaterialVersionBolsaPropuesta, ArchivoMaterialVersionBolsaCierre:
		return false
	}
	return true
}

// escribirPrivadoGobiernoUsuarios crea un fichero 0600 nuevo bajo la raíz
// privada; nunca trunca ni sigue enlaces. Si la escritura falla, lo retira.
func escribirPrivadoGobiernoUsuarios(root *os.Root, nombre string, data []byte) error {
	f, err := reservarPrivadoGobiernoUsuarios(root, nombre)
	if err != nil {
		return err
	}
	if completarPrivadoGobiernoUsuarios(f, data) != nil {
		_ = root.Remove(nombre)
		return ErrGobiernoUsuariosAdmin
	}
	return nil
}

func reservarPrivadoGobiernoUsuarios(root *os.Root, nombre string) (*os.File, error) {
	if root == nil || nombre == "" || nombre == "." || nombre == ".." || filepath.Base(nombre) != nombre {
		return nil, ErrGobiernoUsuariosAdmin
	}
	f, err := root.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	return f, nil
}

// completarPrivadoGobiernoUsuarios escribe, sincroniza y cierra el descriptor
// reservado; un cierre repetido posterior es inocuo.
func completarPrivadoGobiernoUsuarios(f *os.File, data []byte) error {
	if f == nil {
		return ErrGobiernoUsuariosAdmin
	}
	_, err := f.Write(data)
	sincronizar := f.Sync()
	cerrar := f.Close()
	if err != nil || sincronizar != nil || cerrar != nil {
		return ErrGobiernoUsuariosAdmin
	}
	return nil
}

func leerPrivadoGobiernoUsuarios(root *os.Root, nombre string) ([]byte, error) {
	if root == nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	info, err := root.Lstat(nombre)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || !archivoDenominacionPropio(info) || info.Size() < 1 || info.Size() > limiteArchivoGobiernoUsuarios {
		return nil, ErrGobiernoUsuariosAdmin
	}
	f, err := root.OpenFile(nombre, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrGobiernoUsuariosAdmin
	}
	defer f.Close()
	abierto, err := f.Stat()
	if err != nil || !os.SameFile(info, abierto) {
		return nil, ErrGobiernoUsuariosAdmin
	}
	b, err := io.ReadAll(io.LimitReader(f, info.Size()+1))
	if err != nil || int64(len(b)) != info.Size() {
		borrarBytes(b)
		return nil, ErrGobiernoUsuariosAdmin
	}
	return b, nil
}
