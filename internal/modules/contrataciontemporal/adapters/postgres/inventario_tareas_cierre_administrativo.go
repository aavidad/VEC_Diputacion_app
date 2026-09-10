package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// PublicacionLibroCierreAdministrativoPreparada contiene los tres materiales
// reproducibles para la publicación explícita por el propietario. No escribe
// SQL, asigna referencias ni decide cuándo entra en vigor una configuración.
type PublicacionLibroCierreAdministrativoPreparada struct {
	PublicacionJSON []byte
	Canonico        []byte
	HuellaSHA256    string
}

func PrepararPublicacionLibroCierreAdministrativo(p domain.PublicacionLibroTareasCierreAdministrativoEjercicio) (PublicacionLibroCierreAdministrativoPreparada, error) {
	canon, err := canonLibroTareasCierre87(p)
	if err != nil {
		return PublicacionLibroCierreAdministrativoPreparada{}, err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return PublicacionLibroCierreAdministrativoPreparada{}, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	h := sha256.Sum256(canon)
	return PublicacionLibroCierreAdministrativoPreparada{b, canon, hex.EncodeToString(h[:])}, nil
}

const dominioLibroCierre87 = "vec.dipgra.contratacion-temporal.cierre-administrativo.libro"
const dominioSnapshotCierre87 = "vec.dipgra.contratacion-temporal.cierre-administrativo.inventario"

type canonCierre87 struct{ bytes.Buffer }

func nuevoCanonCierre87(dominio string) *canonCierre87 {
	c := &canonCierre87{}
	c.texto(dominio)
	_ = binary.Write(&c.Buffer, binary.BigEndian, uint16(1))
	c.texto("sha-256")
	return c
}
func (c *canonCierre87) texto(s string) {
	_ = binary.Write(&c.Buffer, binary.BigEndian, uint32(len(s)))
	c.WriteString(s)
}
func (c *canonCierre87) numero(n uint64) { _ = binary.Write(&c.Buffer, binary.BigEndian, n) }
func (c *canonCierre87) bloque(b []byte) {
	_ = binary.Write(&c.Buffer, binary.BigEndian, uint32(len(b)))
	c.Write(b)
}

func canonLibroTareasCierre87(p domain.PublicacionLibroTareasCierreAdministrativoEjercicio) ([]byte, error) {
	if p.Validar() != nil || p.Version > 1<<53-1 {
		return nil, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	c := nuevoCanonCierre87(dominioLibroCierre87)
	c.texto(p.Referencia)
	c.numero(p.Version)
	c.texto(p.Ambito)
	_ = binary.Write(&c.Buffer, binary.BigEndian, uint32(len(p.Tareas)))
	for _, t := range p.Tareas {
		c.texto(string(t.Clave))
		c.texto(string(t.Evidencia))
	}
	return c.Bytes(), nil
}

func canonInventarioTareasCierre87(entrada ports.EntradaInventarioTareasCierreAdministrativoEjercicio) ([]byte, error) {
	if _, err := entrada.Preparar(); err != nil {
		return nil, err
	}
	libro, err := canonLibroTareasCierre87(entrada.Libro)
	if err != nil {
		return nil, err
	}
	c := nuevoCanonCierre87(dominioSnapshotCierre87)
	c.bloque(libro)
	c.texto(entrada.OrganizacionRef)
	c.texto(entrada.ExpedienteRef)
	c.texto(entrada.SeguimientoRef)
	_ = binary.Write(&c.Buffer, binary.BigEndian, uint32(len(entrada.Estados)))
	for _, e := range entrada.Estados {
		c.texto(string(e.Clave))
		if e.Pendiente {
			c.WriteByte(1)
		} else {
			c.WriteByte(0)
		}
		v := e.Evidencia
		if v.VersionEvidencia > 1<<53-1 {
			return nil, ports.ErrPreparacionCierreAdministrativoInvalida
		}
		c.texto(string(v.Tipo))
		c.texto(v.Referencia)
		c.texto(v.OrganizacionRef)
		c.texto(v.ExpedienteRef)
		c.texto(v.SeguimientoRef)
		c.numero(v.VersionSeguimientoOriginal)
		c.texto(v.HuellaRaizSeguimientoSHA256)
		c.numero(v.VersionEvidencia)
		c.texto(v.HuellaEvidenciaSHA256)
	}
	return c.Bytes(), nil
}

func restaurarInventarioTareasCierre87(entrada ports.EntradaInventarioTareasCierreAdministrativoEjercicio, canonHex, huella string) (ports.InventarioTareasCierreAdministrativo, error) {
	esperado, err := canonInventarioTareasCierre87(entrada)
	recibido, errHex := hex.DecodeString(canonHex)
	h := sha256.Sum256(esperado)
	if err != nil || errHex != nil || !bytes.Equal(esperado, recibido) || hex.EncodeToString(h[:]) != huella {
		return ports.InventarioTareasCierreAdministrativo{}, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	return entrada.Preparar()
}
