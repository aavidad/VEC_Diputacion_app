package copias

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"strconv"
)

// Las huellas canónicas encuadran cada valor por longitud y ordenan conjuntos.
// No dependen del orden JSON ni del orden de las listas. No autentican un origen.
func valores(h hash.Hash, ss ...string) {
	for _, s := range ss {
		_, _ = fmt.Fprintf(h, "%d:", len(s))
		_, _ = h.Write([]byte(s))
	}
}

func huella(escribir func(hash.Hash)) string {
	h := sha256.New()
	escribir(h)
	return hex.EncodeToString(h.Sum(nil))
}

func huellaArtefactos(as []Artefacto) string {
	cs := append([]Artefacto(nil), as...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return huella(func(h hash.Hash) {
		valores(h, strconv.Itoa(len(cs)))
		for _, a := range cs {
			valores(h, a.ID, a.Tipo, a.SHA256, strconv.FormatInt(a.TamanoBytes, 10))
		}
	})
}

func huellaModulos(ms []Modulo) string {
	cs := append([]Modulo(nil), ms...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return huella(func(h hash.Hash) {
		valores(h, strconv.Itoa(len(cs)))
		for _, m := range cs {
			migs := append([]Migracion(nil), m.Migraciones...)
			sort.Slice(migs, func(i, j int) bool { return migs[i].ID < migs[j].ID })
			valores(h, m.ID, m.EsquemaSHA256, strconv.Itoa(len(migs)))
			for _, mig := range migs {
				valores(h, mig.ID, mig.SHA256)
			}
		}
	})
}

func huellaRuntime(pg RuntimeAdmitido) string {
	hs := append([]Herramienta(nil), pg.Herramientas...)
	sort.Slice(hs, func(i, j int) bool { return hs[i].ID < hs[j].ID })
	es := append([]Extension(nil), pg.Extensiones...)
	sort.Slice(es, func(i, j int) bool { return es[i].ID < es[j].ID })
	return huella(func(h hash.Hash) {
		valores(h, pg.Version, pg.RuntimeSHA256, pg.Plataforma, strconv.Itoa(len(hs)))
		for _, x := range hs {
			valores(h, x.ID, x.Version, x.SHA256)
		}
		valores(h, strconv.Itoa(len(es)))
		for _, x := range es {
			valores(h, x.ID, x.Version)
		}
	})
}

func huellaRelease(r Release) string {
	return huella(func(h hash.Hash) {
		valores(h, r.ID, r.Commit, r.Plataforma, huellaArtefactos(r.Binarios), huellaArtefactos(r.Componentes), huellaModulos(r.EsquemaEsperado))
	})
}

func huellaStrings(ss []string) string {
	cs := append([]string(nil), ss...)
	sort.Strings(cs)
	return huella(func(h hash.Hash) { valores(h, strconv.Itoa(len(cs))); valores(h, cs...) })
}

func huellaAlmacenes(as []Almacen) string {
	cs := append([]Almacen(nil), as...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return huella(func(h hash.Hash) {
		valores(h, strconv.Itoa(len(cs)))
		for _, a := range cs {
			valores(h, a.ID, a.Tipo, a.SHA256)
		}
	})
}

// HuellaInventario permite sellar una declaración canónica. Sus consumidores
// deben verificar además autenticidad y bytes mediante los puertos correspondientes.
func HuellaInventario(i Inventario) string {
	return huella(func(h hash.Hash) {
		valores(h, "vec-copias-inventario-v1", strconv.Itoa(i.FormatoVersion), i.Ref, strconv.FormatBool(i.Completo), huellaRuntime(runtimeDe(i.PostgreSQL)), i.PostgreSQL.ClusterRef, strconv.FormatBool(i.PostgreSQL.AmbitoCompleto), huellaStrings(i.PostgreSQL.Bases), huellaAlmacenes(i.PostgreSQL.Almacenes), huellaRelease(i.Release), huellaModulos(i.Modulos))
	})
}
