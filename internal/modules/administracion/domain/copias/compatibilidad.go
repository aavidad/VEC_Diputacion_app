package copias

import (
	"fmt"
	"hash"
	"sort"
	"strconv"
)

type comparacion struct{ razones []Razon }

func (c *comparacion) igual(ok bool, codigo, clave, esperado, obtenido, accion string) {
	if !ok {
		c.razones = append(c.razones, Razon{codigo, clave, esperado, obtenido, accion})
	}
}

func (c *comparacion) modulos(a, b []Modulo, clave string) {
	c.igual(len(a) == len(b), "modulos_diferentes", clave+".cantidad", strconv.Itoa(len(a)), strconv.Itoa(len(b)), "preparar_conjunto_completo")
	c.igual(huellaModulos(a) == huellaModulos(b), "esquema_migraciones_diferentes", clave, huellaModulos(a), huellaModulos(b), "preparar_actualizacion_separada")
	cs := append([]Modulo(nil), a...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	for i, m := range cs {
		var observado *Modulo
		for j := range b {
			if b[j].ID == m.ID {
				observado = &b[j]
				break
			}
		}
		if observado == nil {
			continue
		}
		c.igual(m.EsquemaSHA256 == observado.EsquemaSHA256, "esquema_huella_diferente", fmt.Sprintf("%s.modulos[%d].esquema_sha256", clave, i), m.EsquemaSHA256, observado.EsquemaSHA256, "preparar_actualizacion_separada")
		migs := append([]Migracion(nil), m.Migraciones...)
		sort.Slice(migs, func(i, j int) bool { return migs[i].ID < migs[j].ID })
		for j, mig := range migs {
			for _, x := range observado.Migraciones {
				if mig.ID == x.ID {
					c.igual(mig.SHA256 == x.SHA256, "migracion_huella_diferente", fmt.Sprintf("%s.modulos[%d].migraciones[%d].sha256", clave, i, j), mig.SHA256, x.SHA256, "preparar_actualizacion_separada")
				}
			}
		}
	}
}

func huellaValor(s string) string { return huella(func(h hash.Hash) { valores(h, s) }) }

func (c *comparacion) release(a, b Release, clave string) {
	c.igual(a.ID == b.ID, "release_diferente", clave+".id", huellaValor(a.ID), huellaValor(b.ID), "recuperar_release_archivada")
	c.igual(a.Commit == b.Commit, "commit_diferente", clave+".commit", a.Commit, b.Commit, "recuperar_release_archivada")
	c.igual(a.Plataforma == b.Plataforma, "plataforma_diferente", clave+".plataforma", huellaValor(a.Plataforma), huellaValor(b.Plataforma), "preparar_destino_compatible")
	c.igual(huellaArtefactos(a.Binarios) == huellaArtefactos(b.Binarios), "binarios_diferentes", clave+".binarios", huellaArtefactos(a.Binarios), huellaArtefactos(b.Binarios), "recuperar_release_archivada")
	c.igual(huellaArtefactos(a.Componentes) == huellaArtefactos(b.Componentes), "componentes_diferentes", clave+".componentes", huellaArtefactos(a.Componentes), huellaArtefactos(b.Componentes), "recuperar_conjunto_completo")
	c.modulos(a.EsquemaEsperado, b.EsquemaEsperado, clave+".esquema_esperado")
}

func (c *comparacion) runtime(a, b RuntimeAdmitido, clave string) {
	c.igual(a.Version == b.Version, "postgresql_version_diferente", clave+".version", a.Version, b.Version, "preparar_runtime_exacto")
	c.igual(huellaRuntime(a) == huellaRuntime(b), "runtime_diferente", clave, huellaRuntime(a), huellaRuntime(b), "preparar_runtime_exacto")
}

func (c *comparacion) resultado() Resultado {
	estado := Compatible
	if len(c.razones) > 0 {
		estado = Incompatible
	}
	return Resultado{estado, ordenar(c.razones)}
}

// CompararInventarios sirve antes de copiar: contrasta la instalación observada
// con su descriptor. No construye evidencia de copia ni ejecuta ningún efecto.
func CompararInventarios(esperado, observado Inventario) Resultado {
	rs := append(ValidarInventario(esperado), ValidarInventario(observado)...)
	if len(rs) > 0 {
		return Resultado{NoComprobable, ordenar(rs)}
	}
	c := comparacion{}
	c.runtime(runtimeDe(esperado.PostgreSQL), runtimeDe(observado.PostgreSQL), "inventario.postgresql")
	c.igual(esperado.PostgreSQL.ClusterRef == observado.PostgreSQL.ClusterRef, "cluster_diferente", "inventario.postgresql.cluster_ref", huellaValor(esperado.PostgreSQL.ClusterRef), huellaValor(observado.PostgreSQL.ClusterRef), "comprobar_origen")
	c.igual(huellaStrings(esperado.PostgreSQL.Bases) == huellaStrings(observado.PostgreSQL.Bases), "bases_diferentes", "inventario.postgresql.bases", huellaStrings(esperado.PostgreSQL.Bases), huellaStrings(observado.PostgreSQL.Bases), "completar_inventario")
	c.igual(huellaAlmacenes(esperado.PostgreSQL.Almacenes) == huellaAlmacenes(observado.PostgreSQL.Almacenes), "almacenes_diferentes", "inventario.postgresql.almacenes", huellaAlmacenes(esperado.PostgreSQL.Almacenes), huellaAlmacenes(observado.PostgreSQL.Almacenes), "completar_inventario")
	c.release(esperado.Release, observado.Release, "inventario.release")
	c.modulos(esperado.Modulos, observado.Modulos, "inventario.modulos")
	c.modulos(esperado.Release.EsquemaEsperado, esperado.Modulos, "descriptor.esquema_instalado")
	c.modulos(observado.Release.EsquemaEsperado, observado.Modulos, "observado.esquema_instalado")
	return c.resultado()
}

// CompararVersiones compara la combinación FINAL de V1: el conjunto entero de
// la copia, incluido su binario. El destino actual puede tener otra release.
// Compatible se refiere solo a declaraciones: nunca habilita restauraciones.
func CompararVersiones(m Manifiesto, destino Inventario, p Politica, modo ModoRestauracion) Resultado {
	if modo != ConjuntoCompleto {
		return Resultado{Incompatible, []Razon{{"modo_no_admitido", "restauracion.modo", huellaValor(string(ConjuntoCompleto)), huellaValor(string(modo)), "restaurar_conjunto_completo"}}}
	}
	rs := append(ValidarManifiesto(m), ValidarInventario(destino)...)
	rs = append(rs, ValidarPolitica(p)...)
	if len(rs) > 0 {
		return Resultado{NoComprobable, ordenar(rs)}
	}
	c := comparacion{}
	c.igual(m.InventarioSHA256 == HuellaInventario(m.Inventario), "inventario_huella_diferente", "manifiesto.inventario_sha256", m.InventarioSHA256, HuellaInventario(m.Inventario), "comprobar_integridad")
	c.modulos(m.Inventario.Release.EsquemaEsperado, m.Inventario.Modulos, "copia.esquema_instalado")
	c.modulos(destino.Release.EsquemaEsperado, destino.Modulos, "destino.esquema_instalado")
	c.runtime(runtimeDe(m.Inventario.PostgreSQL), runtimeDe(destino.PostgreSQL), "destino.postgresql")
	c.igual(m.Inventario.Release.Plataforma == destino.Release.Plataforma, "plataforma_diferente", "destino.release.plataforma", huellaValor(m.Inventario.Release.Plataforma), huellaValor(destino.Release.Plataforma), "preparar_destino_compatible")
	c.igual(huellaStrings(m.Inventario.PostgreSQL.Bases) == huellaStrings(destino.PostgreSQL.Bases), "bases_diferentes", "destino.postgresql.bases", huellaStrings(m.Inventario.PostgreSQL.Bases), huellaStrings(destino.PostgreSQL.Bases), "preparar_cluster_dedicado")
	// El cluster_ref puede diferir: es otro cluster dedicado. Sus almacenes deben
	// estar declarados por la misma identidad/tipo; el contenido será sustituido.
	c.igual(huellaAlmacenesIdentidad(m.Inventario.PostgreSQL.Almacenes) == huellaAlmacenesIdentidad(destino.PostgreSQL.Almacenes), "almacenes_diferentes", "destino.postgresql.almacenes", huellaAlmacenesIdentidad(m.Inventario.PostgreSQL.Almacenes), huellaAlmacenesIdentidad(destino.PostgreSQL.Almacenes), "preparar_destino_compatible")
	archivados := append(append([]Artefacto(nil), m.Inventario.Release.Binarios...), m.Inventario.Release.Componentes...)
	var observados []Artefacto
	for _, a := range archivados {
		for _, b := range m.Componentes {
			if a.ID == b.ID {
				observados = append(observados, b)
			}
		}
	}
	c.igual(huellaArtefactos(archivados) == huellaArtefactos(observados), "release_incompleta", "manifiesto.componentes", huellaArtefactos(archivados), huellaArtefactos(observados), "recuperar_conjunto_completo")
	admitido := false
	var runtimesAdmitidos []string
	for _, r := range p.Runtimes {
		runtimesAdmitidos = append(runtimesAdmitidos, huellaRuntime(r))
		admitido = admitido || huellaRuntime(r) == huellaRuntime(runtimeDe(m.Inventario.PostgreSQL))
	}
	c.igual(admitido, "runtime_no_admitido", "politica.runtimes", huellaStrings(runtimesAdmitidos), huellaRuntime(runtimeDe(m.Inventario.PostgreSQL)), "actualizar_politica_o_destino")
	admitido = false
	var releasesAdmitidas []string
	for _, r := range p.Releases {
		releasesAdmitidas = append(releasesAdmitidas, huellaRelease(r))
		admitido = admitido || huellaRelease(r) == huellaRelease(m.Inventario.Release)
	}
	c.igual(admitido, "release_no_admitida", "politica.releases", huellaStrings(releasesAdmitidas), huellaRelease(m.Inventario.Release), "preparar_actualizacion_separada")
	for _, id := range p.ReleasesRevocadas {
		c.igual(id != m.Inventario.Release.ID, "release_revocada", "politica.releases_revocadas", "false", "true", "preparar_release_vigente")
	}
	return c.resultado()
}

func huellaAlmacenesIdentidad(as []Almacen) string {
	cs := make([]Almacen, len(as))
	for i, a := range as {
		cs[i] = Almacen{ID: a.ID, Tipo: a.Tipo}
	}
	return huellaAlmacenes(cs)
}
