package copias

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	idRE      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_.-]{0,127}$`)
	shaRE     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitRE  = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
	versionRE = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}$`)
)

type validacion struct{ razones []Razon }

func (v *validacion) exigir(ok bool, clave, esperado, obtenido string) {
	if !ok {
		v.razones = append(v.razones, Razon{"dato_no_comprobable", clave, esperado, obtenido, "completar_inventario"})
	}
}

func (v *validacion) id(s, clave string) {
	v.exigir(idRE.MatchString(s), clave, "referencia_opaca", "ausente_o_invalida")
}
func (v *validacion) sha(s, clave string) {
	v.exigir(shaRE.MatchString(s), clave, "sha256", "ausente_o_invalida")
}
func (v *validacion) formato(n int, clave string) {
	v.exigir(n == FormatoVersion, clave, strconv.Itoa(FormatoVersion), strconv.Itoa(n))
}

func (v *validacion) conjunto(ids []string, clave string, noVacio bool) {
	v.exigir(!noVacio || len(ids) > 0, clave, "conjunto_no_vacio", "vacio")
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		v.id(id, clave)
		v.exigir(!seen[id], clave, "identificadores_unicos", "duplicado")
		seen[id] = true
	}
}

func (v *validacion) artefactos(as []Artefacto, clave string, noVacio bool) {
	ids := make([]string, 0, len(as))
	for _, a := range as {
		ids = append(ids, a.ID)
		v.id(a.Tipo, clave+".tipo")
		v.sha(a.SHA256, clave+".sha256")
		v.exigir(a.TamanoBytes >= 0, clave+".tamano_bytes", "no_negativo", "negativo")
	}
	v.conjunto(ids, clave, noVacio)
}

func (v *validacion) modulos(ms []Modulo, clave string) {
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		ids = append(ids, m.ID)
		v.sha(m.EsquemaSHA256, clave+".esquema_sha256")
		migIDs := make([]string, 0, len(m.Migraciones))
		for _, mig := range m.Migraciones {
			migIDs = append(migIDs, mig.ID)
			v.sha(mig.SHA256, clave+".migraciones.sha256")
		}
		v.conjunto(migIDs, clave+".migraciones", false)
	}
	v.conjunto(ids, clave, true)
}

func (v *validacion) release(r Release, clave string) {
	v.id(r.ID, clave+".id")
	v.exigir(commitRE.MatchString(r.Commit), clave+".commit", "commit_completo", "ausente_o_invalido")
	v.id(r.Plataforma, clave+".plataforma")
	v.artefactos(r.Binarios, clave+".binarios", true)
	v.artefactos(r.Componentes, clave+".componentes", true)
	ids := make([]string, 0, len(r.Binarios)+len(r.Componentes))
	for _, a := range r.Binarios {
		ids = append(ids, a.ID)
		v.exigir(a.Tipo == "binario", clave+".binarios.tipo", "binario", "otro_tipo")
	}
	for _, a := range r.Componentes {
		ids = append(ids, a.ID)
	}
	v.conjunto(ids, clave+".artefactos", true)
	for _, tipo := range []string{"web", "catalogos", "material", "configuracion", "ficheros"} {
		found := false
		for _, a := range r.Componentes {
			found = found || a.Tipo == tipo
		}
		v.exigir(found, clave+".componentes."+tipo, "inventariado", "ausente")
	}
	v.modulos(r.EsquemaEsperado, clave+".esquema_esperado")
}

func (v *validacion) runtime(pg RuntimeAdmitido, clave string) {
	v.exigir(versionRE.MatchString(pg.Version), clave+".version", "version_exacta", "ausente_o_invalida")
	v.sha(pg.RuntimeSHA256, clave+".runtime_sha256")
	v.id(pg.Plataforma, clave+".plataforma")
	ids := make([]string, 0, len(pg.Herramientas))
	for _, h := range pg.Herramientas {
		ids = append(ids, h.ID)
		v.exigir(versionRE.MatchString(h.Version), clave+".herramientas.version", "version_exacta", "ausente_o_invalida")
		v.sha(h.SHA256, clave+".herramientas.sha256")
	}
	v.conjunto(ids, clave+".herramientas", true)
	for _, id := range []string{"pg_dump", "pg_restore", "psql", "postgres"} {
		found := false
		for _, h := range pg.Herramientas {
			found = found || h.ID == id
		}
		v.exigir(found, clave+".herramientas."+id, "inventariada", "ausente")
	}
	ids = make([]string, 0, len(pg.Extensiones))
	for _, e := range pg.Extensiones {
		ids = append(ids, e.ID)
		v.id(e.Version, clave+".extensiones.version")
	}
	v.conjunto(ids, clave+".extensiones", false)
}

func runtimeDe(pg PostgreSQL) RuntimeAdmitido {
	return RuntimeAdmitido{pg.Version, pg.RuntimeSHA256, pg.Plataforma, pg.Herramientas, pg.Extensiones}
}

func (v *validacion) inventario(i Inventario, clave string) {
	v.formato(i.FormatoVersion, clave+".formato_version")
	v.id(i.Ref, clave+".ref")
	v.exigir(i.Completo, clave+".completo", "completo", "incompleto")
	v.runtime(runtimeDe(i.PostgreSQL), clave+".postgresql")
	v.id(i.PostgreSQL.ClusterRef, clave+".postgresql.cluster_ref")
	v.conjunto(i.PostgreSQL.Bases, clave+".postgresql.bases", true)
	v.exigir(i.PostgreSQL.AmbitoCompleto, clave+".postgresql.ambito_completo", "cluster_autorizado_completo", "no_acreditado")
	ids := make([]string, 0, len(i.PostgreSQL.Almacenes))
	for _, a := range i.PostgreSQL.Almacenes {
		ids = append(ids, a.ID)
		v.id(a.Tipo, clave+".postgresql.almacenes.tipo")
		v.sha(a.SHA256, clave+".postgresql.almacenes.sha256")
	}
	v.conjunto(ids, clave+".postgresql.almacenes", false)
	v.release(i.Release, clave+".release")
	v.modulos(i.Modulos, clave+".modulos")
}

// ValidarInventario valida estructura y cobertura declarada, sin leer ficheros.
func ValidarInventario(i Inventario) []Razon {
	v := validacion{}
	v.inventario(i, "inventario")
	return ordenar(v.razones)
}

func (v *validacion) evidencia(e Evidencia, clave string) {
	v.id(e.Ref, clave+".ref")
	v.id(e.ArranqueRef, clave+".arranque_ref")
	for _, item := range []struct{ k, s string }{{"recuentos", e.RecuentosSHA256}, {"contenido", e.ContenidoSHA256}, {"esquema", e.EsquemaSHA256}, {"roles", e.RolesSHA256}, {"acl", e.ACLSHA256}, {"secuencias", e.SecuenciasSHA256}, {"objetos_grandes", e.ObjetosGrandesSHA256}, {"ficheros", e.FicherosSHA256}} {
		v.sha(item.s, clave+"."+item.k+"_sha256")
	}
}

// ValidarManifiesto comprueba metadatos e inventarios. No verifica sus bytes,
// firmas, cifrado ni referencias de restauración.
func ValidarManifiesto(m Manifiesto) []Razon {
	v := validacion{}
	v.formato(m.FormatoVersion, "manifiesto.formato_version")
	for _, item := range []struct{ k, s string }{{"conjunto_ref", m.ConjuntoRef}, {"operacion_ref", m.OperacionRef}, {"solicitante_ref", m.SolicitanteRef}, {"motivo_ref", m.MotivoRef}, {"politica_ref", m.PoliticaRef}} {
		v.id(item.s, "manifiesto."+item.k)
	}
	_, inicioOffset := m.Inicio.Zone()
	_, finOffset := m.Fin.Zone()
	v.exigir(!m.Inicio.IsZero() && !m.Fin.IsZero() && inicioOffset == 0 && finOffset == 0 && !m.Fin.Before(m.Inicio), "manifiesto.fechas", "intervalo_utc_valido", "ausente_o_invalido")
	v.exigir(m.TamanoBytes >= 0, "manifiesto.tamano_bytes", "no_negativo", "negativo")
	v.inventario(m.Inventario, "manifiesto.inventario")
	v.artefactos(m.Componentes, "manifiesto.componentes", true)
	v.sha(m.InventarioSHA256, "manifiesto.inventario_sha256")
	for _, tipo := range []string{"base_fisica", "base_logica", "globals", "binario", "web", "catalogos", "material", "configuracion", "ficheros"} {
		found := false
		for _, a := range m.Componentes {
			found = found || a.Tipo == tipo
		}
		v.exigir(found, "manifiesto.componentes."+tipo, "inventariado", "ausente")
	}
	var total int64
	for _, a := range m.Componentes {
		// Detecta desbordamiento antes de sumar tamaños controlados por la entrada.
		if a.TamanoBytes < 0 || total > int64(^uint64(0)>>1)-a.TamanoBytes {
			v.exigir(false, "manifiesto.tamano_bytes", "suma_valida", "fuera_de_rango")
			break
		}
		total += a.TamanoBytes
	}
	v.exigir(total == m.TamanoBytes, "manifiesto.tamano_bytes", "suma_componentes", "total_distinto")
	v.exigir(m.Consistencia.Modo == "fisica_fria_y_logica", "manifiesto.consistencia.modo", "fisica_fria_y_logica", "otro_modo")
	v.exigir(m.Consistencia.EscritoresExcluidos && m.Consistencia.ParadaLimpia, "manifiesto.consistencia", "exclusion_y_parada_limpia", "no_acreditadas")
	v.id(m.Consistencia.EvidenciaRef, "manifiesto.consistencia.evidencia_ref")
	v.id(m.Consistencia.PuntoRecuperacionRef, "manifiesto.consistencia.punto_recuperacion_ref")
	for _, item := range []struct{ k, s string }{{"formato", m.Proteccion.Formato}, {"algoritmo", m.Proteccion.Algoritmo}, {"clave_ref", m.Proteccion.ClaveRef}, {"clave_version", m.Proteccion.ClaveVersion}, {"autenticacion_ref", m.Proteccion.AutenticacionRef}} {
		v.id(item.s, "manifiesto.proteccion."+item.k)
	}
	v.sha(m.Proteccion.CifradoSHA256, "manifiesto.proteccion.cifrado_sha256")
	v.exigir(m.Verificacion.Estado == "valida" || m.Verificacion.Estado == "no_valida" || m.Verificacion.Estado == "pendiente_verificacion", "manifiesto.verificacion.estado", "estado_conocido", "desconocido")
	if m.Verificacion.Estado == "valida" || m.Verificacion.Estado == "no_valida" {
		v.id(m.Verificacion.VerificadorVersion, "manifiesto.verificacion.verificador_version")
		_, off := m.Verificacion.Fecha.Zone()
		v.exigir(!m.Verificacion.Fecha.IsZero() && off == 0 && !m.Verificacion.Fecha.Before(m.Fin), "manifiesto.verificacion.fecha", "fecha_utc_posterior", "ausente_o_invalida")
		v.evidencia(m.Verificacion.Fisica, "manifiesto.verificacion.fisica")
		v.evidencia(m.Verificacion.Logica, "manifiesto.verificacion.logica")
	}
	return ordenar(v.razones)
}

func ValidarPolitica(p Politica) []Razon {
	v := validacion{}
	v.formato(p.FormatoVersion, "politica.formato_version")
	v.id(p.Ref, "politica.ref")
	ids := make([]string, 0, len(p.Runtimes))
	for _, r := range p.Runtimes {
		v.runtime(r, "politica.runtimes")
		ids = append(ids, huellaRuntime(r))
	}
	v.conjunto(ids, "politica.runtimes", true)
	ids = make([]string, 0, len(p.Releases))
	for _, r := range p.Releases {
		v.release(r, "politica.releases")
		ids = append(ids, r.ID)
	}
	v.conjunto(ids, "politica.releases", true)
	v.conjunto(p.ReleasesRevocadas, "politica.releases_revocadas", false)
	return ordenar(v.razones)
}

func ordenar(rs []Razon) []Razon {
	if rs == nil {
		return []Razon{}
	}
	sort.Slice(rs, func(i, j int) bool {
		a, b := rs[i], rs[j]
		return strings.Join([]string{a.Clave, a.Codigo, a.Esperado, a.Obtenido, a.Accion}, "\x00") < strings.Join([]string{b.Clave, b.Codigo, b.Esperado, b.Obtenido, b.Accion}, "\x00")
	})
	return rs
}
