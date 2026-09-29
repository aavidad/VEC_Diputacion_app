// Package application coordina los casos de uso de la ficha propia de la
// persona aspirante: consultar, darse de alta y rectificar su contacto.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"time"

	"vec-diputacion-granada/internal/modules/aspirantes/canonico"
	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Dependencias agrupa los puertos del servicio. Ninguno es opcional.
type Dependencias struct {
	Registro  ports.RegistroFichas
	Protector ports.ProtectorFicha
	Sellador  ports.SelladorHuella
	Catalogo  ports.CatalogoExigenciasFicha
	Azar      io.Reader
	AhoraUTC  func() time.Time
}

type ServicioFichaPropia struct{ d Dependencias }

func NuevoServicioFichaPropia(d Dependencias) (*ServicioFichaPropia, error) {
	if d.Registro == nil || d.Protector == nil || d.Sellador == nil || d.Catalogo == nil || d.Azar == nil || d.AhoraUTC == nil {
		return nil, ports.ErrNoDisponible
	}
	return &ServicioFichaPropia{d: d}, nil
}

// NuevaOrdenFicha liga la sesión del portal externo, la identidad acreditada
// por el mismo certificado y el puerto V3. La ruta aporta la superficie.
func NuevaOrdenFicha(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1, identidad domain.IdentidadAcreditada, proveedor ports.ProveedorMaterialFicha) (ports.OrdenFicha, error) {
	if proveedor == nil {
		return ports.OrdenFicha{}, ports.ErrNoAutenticado
	}
	sesion, err := domain.NuevaSesionAspirante(actor, vinculo, superficieRuta)
	if errors.Is(err, domain.ErrSuperficieProhibida) {
		return ports.OrdenFicha{}, ports.ErrProhibido
	}
	if err != nil || identidad.Validar() != nil {
		return ports.OrdenFicha{}, ports.ErrNoAutenticado
	}
	return ports.OrdenFicha{Sesion: sesion, Identidad: identidad, Proveedor: proveedor}, nil
}

// Vistas que devuelve el servicio. Nunca incluyen `asp_`, `per_` ni el
// número de documento completo.
type VistaIdentidad struct {
	Nombre          string `json:"nombre"`
	PrimerApellido  string `json:"primer_apellido"`
	SegundoApellido string `json:"segundo_apellido"`
	TipoDocumento   string `json:"tipo_documento"`
	PaisDocumento   string `json:"pais_documento"`
	Documento       string `json:"documento"`
}

type VistaExigencia struct {
	Campo       string `json:"campo"`
	Obligatorio bool   `json:"obligatorio"`
}

const (
	EstadoSinFicha = "sin_ficha"
	EstadoActiva   = "activa"
)

type VistaFichaPropia struct {
	Estado             string            `json:"estado"`
	Version            uint64            `json:"version"`
	Identidad          VistaIdentidad    `json:"identidad"`
	Contacto           map[string]string `json:"contacto"`
	Exigencias         []VistaExigencia  `json:"exigencias"`
	CatalogoDisponible bool              `json:"catalogo_disponible"`
}

// PeticionFicha es lo único que aporta el navegador. La persona, el perfil,
// la superficie y la identidad salen de la sesión y del certificado.
type PeticionFicha struct {
	ClaveOperacion  string
	VersionEsperada uint64
	Motivo          string
	Campos          map[string]string
}

func (s *ServicioFichaPropia) actor(ctx context.Context, orden ports.OrdenFicha) (vecdomain.ContextoActor, error) {
	if s == nil || s.d.Registro == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, ports.ErrNoDisponible
	}
	if orden.Proveedor == nil || orden.Identidad.Validar() != nil {
		return vecdomain.ContextoActor{}, ports.ErrNoAutenticado
	}
	actor, vinculo, err := orden.Sesion.Datos()
	if errors.Is(err, domain.ErrSuperficieProhibida) {
		return vecdomain.ContextoActor{}, ports.ErrProhibido
	}
	if err != nil {
		return vecdomain.ContextoActor{}, ports.ErrNoAutenticado
	}
	ahora := s.d.AhoraUTC().UTC().Truncate(time.Microsecond)
	datos, err := vinculo.Datos()
	if err != nil || ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) || ahora.Before(datos.SesionRevalidadaEn) || !ahora.Before(datos.SesionValidaHasta) {
		return vecdomain.ContextoActor{}, ports.ErrNoAutenticado
	}
	return actor, nil
}

func (s *ServicioFichaPropia) indice(ctx context.Context, orden ports.OrdenFicha) (ports.IndiceDocumento, error) {
	indice, err := s.d.Protector.IndiceDocumento(ctx, orden.Identidad.Documento())
	if err != nil || !canonico.IndiceValido(indice) {
		return ports.IndiceDocumento{}, ports.ErrNoDisponible
	}
	return indice, nil
}

func material(actor vecdomain.ContextoActor, accion string, version uint64, clave string, huellas ports.HuellasSemanticas, indice ports.IndiceDocumento) ports.MaterialFicha {
	return ports.MaterialFicha{
		Superficie: vecdomain.SuperficieAutenticacionExternaPersonalV1, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		Accion: accion, FinalidadRef: ports.FinalidadFicha, VersionEsperada: version, ClaveOperacion: clave,
		HuellasPeticion: huellas, IndiceDocumento: indice,
	}
}

// autorizar obtiene una V3 fresca y comprueba que está ligada al material
// exacto: operación, persona, audiencia y huella del recurso.
func autorizar(ctx context.Context, orden ports.OrdenFicha, m ports.MaterialFicha) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	_, vinculo, err := orden.Sesion.Datos()
	if err != nil || orden.Proveedor == nil {
		return vacia, ports.ErrNoAutenticado
	}
	audiencia, err := ports.Audiencia(m.Accion)
	if err != nil {
		return vacia, err
	}
	recurso, err := canonico.Recurso(m)
	if err != nil {
		return vacia, ports.ErrInvalida
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, ports.ErrInvalida
	}
	v3, err := orden.Proveedor.ProveerMaterialFicha(ctx, vinculo, m)
	if err != nil {
		if errors.Is(err, ports.ErrNoAutenticado) || errors.Is(err, ports.ErrProhibido) {
			return vacia, err
		}
		return vacia, ports.ErrNoDisponible
	}
	if v3.ValidarEstructura() != nil {
		return vacia, ports.ErrNoDisponible
	}
	r := v3.ResumenCapacidad()
	if r.Operacion() != m.Accion || r.EfectoRef() != m.PersonaRef || r.AudienciaConsumo() != audiencia || r.EfectoHuellaSHA256() != huella {
		return vacia, ports.ErrNoDisponible
	}
	return v3, nil
}

// nominal deja pasar los errores con significado para la API y convierte
// cualquier otro en «no disponible», sin detalles internos.
func nominal(err error) error {
	for _, e := range []error{ports.ErrNoAutenticado, ports.ErrProhibido, ports.ErrInvalida, ports.ErrConflicto, ports.ErrFichaExistente, ports.ErrSinFicha} {
		if errors.Is(err, e) {
			return e
		}
	}
	return ports.ErrNoDisponible
}

func (s *ServicioFichaPropia) exigencias(ctx context.Context) (ports.ExigenciasContacto, error) {
	e, err := s.d.Catalogo.ExigenciasContactoFichaPropia(ctx)
	if err != nil || domain.ValidarExigencias(e.Campos) != nil || !catalogoRefValida(e.CatalogoRef) {
		return ports.ExigenciasContacto{}, ports.ErrNoDisponible
	}
	return e, nil
}

func catalogoRefValida(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ':' && r != '.' && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func vistaExigencias(e []domain.ExigenciaCampo) []VistaExigencia {
	r := make([]VistaExigencia, 0, len(e))
	for _, x := range e {
		r = append(r, VistaExigencia{Campo: string(x.Campo), Obligatorio: x.Obligatorio})
	}
	return r
}

func vistaIdentidad(valores map[domain.CampoFicha]string, doc domain.DocumentoIdentidad) VistaIdentidad {
	return VistaIdentidad{
		Nombre: valores[domain.CampoNombre], PrimerApellido: valores[domain.CampoPrimerApellido], SegundoApellido: valores[domain.CampoSegundoApellido],
		TipoDocumento: string(doc.Tipo), PaisDocumento: doc.Pais, Documento: doc.Enmascarado(),
	}
}

func borrar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Consultar devuelve la ficha propia o, si aún no existe, la identidad del
// certificado para que la persona la vea antes de crearla (eso no se guarda).
func (s *ServicioFichaPropia) Consultar(ctx context.Context, orden ports.OrdenFicha) (VistaFichaPropia, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return VistaFichaPropia{}, err
	}
	indice, err := s.indice(ctx, orden)
	if err != nil {
		return VistaFichaPropia{}, err
	}
	m := material(actor, ports.AccionConsultar, 0, "", ports.HuellasSemanticas{}, indice)
	v3, err := autorizar(ctx, orden, m)
	if err != nil {
		return VistaFichaPropia{}, err
	}
	ficha, existe, err := s.d.Registro.ConsultarPropia(ctx, orden, m, v3)
	if err != nil {
		return VistaFichaPropia{}, nominal(err)
	}
	vista := VistaFichaPropia{Contacto: map[string]string{}, Exigencias: []VistaExigencia{}}
	if e, err := s.exigencias(ctx); err == nil {
		vista.Exigencias, vista.CatalogoDisponible = vistaExigencias(e.Campos), true
	}
	if !existe {
		vista.Estado = EstadoSinFicha
		vista.Identidad = vistaIdentidad(orden.Identidad.Valores(), orden.Identidad.Documento())
		return vista, nil
	}
	valores, doc, err := s.descifrarFicha(ctx, ficha, indice, orden.Identidad.Documento())
	if err != nil {
		return VistaFichaPropia{}, err
	}
	vista.Estado, vista.Version = EstadoActiva, ficha.Version
	vista.Identidad = vistaIdentidad(valores, doc)
	for campo, valor := range valores {
		if campo.EsContacto() {
			vista.Contacto[string(campo)] = valor
		}
	}
	return vista, nil
}

// descifrarFicha comprueba la forma de lo que devuelve el registro y que el
// documento descifrado es el mismo que acredita el certificado.
func (s *ServicioFichaPropia) descifrarFicha(ctx context.Context, f ports.FichaCifrada, indice ports.IndiceDocumento, esperado domain.DocumentoIdentidad) (map[domain.CampoFicha]string, domain.DocumentoIdentidad, error) {
	vacio := domain.DocumentoIdentidad{}
	if !domain.ReferenciaAspiranteValida(f.AspiranteRef) || f.Version == 0 || f.Version > math.MaxInt64 || f.Documento.Indice != indice ||
		!domain.ReferenciaDocumentoValida(f.Documento.DocumentoRef) || f.Documento.Tipo != esperado.Tipo || f.Documento.Pais != esperado.Pais {
		return nil, vacio, ports.ErrNoDisponible
	}
	claro, err := s.d.Protector.DescifrarDocumento(ctx, f.AspiranteRef, f.Documento.DocumentoRef, f.Documento.Sobre)
	if err != nil {
		return nil, vacio, ports.ErrNoDisponible
	}
	doc, err := domain.NuevoDocumentoIdentidad(f.Documento.Tipo, f.Documento.Pais, string(claro))
	borrar(claro)
	if err != nil || doc != esperado {
		return nil, vacio, ports.ErrNoDisponible
	}
	valores := map[domain.CampoFicha]string{}
	for _, v := range f.Valores {
		if !v.Campo.Valido() || v.Version == 0 || v.Version > f.Version || v.Origen != v.Campo.OrigenEsperado() {
			return nil, vacio, ports.ErrNoDisponible
		}
		if _, repetido := valores[v.Campo]; repetido {
			return nil, vacio, ports.ErrNoDisponible
		}
		if v.Sobre == nil {
			continue
		}
		claro, err := s.d.Protector.DescifrarValor(ctx, f.AspiranteRef, v.Campo, v.Version, *v.Sobre)
		if err != nil {
			return nil, vacio, ports.ErrNoDisponible
		}
		texto := string(claro)
		borrar(claro)
		if normal, err := domain.NormalizarValor(v.Campo, texto); err != nil || normal != texto {
			return nil, vacio, ports.ErrNoDisponible
		}
		valores[v.Campo] = texto
	}
	if valores[domain.CampoNombre] == "" || valores[domain.CampoPrimerApellido] == "" {
		return nil, vacio, ports.ErrNoDisponible
	}
	return valores, doc, nil
}

func claveValida(clave string) bool {
	if len(clave) < 16 || len(clave) > 128 {
		return false
	}
	for _, r := range clave {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != ':' && r != '.' {
			return false
		}
	}
	return true
}

func camposPeticion(p PeticionFicha) (map[domain.CampoFicha]string, error) {
	if len(p.Campos) > len(domain.CamposContacto) {
		return nil, ports.ErrInvalida
	}
	r := make(map[domain.CampoFicha]string, len(p.Campos))
	for k, v := range p.Campos {
		campo := domain.CampoFicha(k)
		if !campo.EsContacto() || len(v) > 1024 {
			return nil, ports.ErrInvalida
		}
		r[campo] = v
	}
	return r, nil
}

// preimagen solo vive en memoria durante el sellado: los valores en claro
// nunca tienen un SHA-256 público que permita un diccionario.
func preimagen(persona, accion string, version uint64, motivo domain.MotivoCambio, catalogo string, indice ports.IndiceDocumento, cambios map[domain.CampoFicha]string) ([]byte, error) {
	pares := make([][2]string, 0, len(cambios))
	for _, c := range domain.CamposOrdenados(cambios) {
		pares = append(pares, [2]string{string(c), cambios[c]})
	}
	b, err := json.Marshal(struct {
		Esquema, Persona, Accion, Motivo, Catalogo, Indice string
		Version                                            uint64
		Campos                                             [][2]string
	}{"aspirantes.ficha.peticion.v1", persona, accion, string(motivo), catalogo, indice.Valor, version, pares})
	if err != nil {
		return nil, ports.ErrInvalida
	}
	return b, nil
}

func (s *ServicioFichaPropia) sellar(ctx context.Context, b []byte) (ports.HuellasSemanticas, error) {
	h, err := s.d.Sellador.SellarHuella(ctx, b)
	borrar(b)
	if err != nil || !canonico.HuellasValidas(h) {
		return ports.HuellasSemanticas{}, ports.ErrNoDisponible
	}
	return h, nil
}

func reciboValido(r ports.ReciboFicha, accion string, version uint64) bool {
	return r.ReciboRef != "" && r.Accion == accion && r.Version == version && !r.FechaUTC.IsZero()
}

// Alta crea la ficha con la identidad del certificado y, si la persona los
// aporta, los datos de contacto que pida el catálogo.
func (s *ServicioFichaPropia) Alta(ctx context.Context, orden ports.OrdenFicha, p PeticionFicha) (ports.ReciboFicha, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	campos, err := camposPeticion(p)
	if err != nil || !claveValida(p.ClaveOperacion) || p.VersionEsperada != 0 || p.Motivo != "" {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	exig, err := s.exigencias(ctx)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	contacto, err := domain.ContactoPedido(campos, exig.Campos, true)
	if err != nil {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	indice, err := s.indice(ctx, orden)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	pre, err := preimagen(actor.PersonaRef, ports.AccionAlta, 0, domain.MotivoAltaTitular, exig.CatalogoRef, indice, contacto)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	huellas, err := s.sellar(ctx, pre)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	m := material(actor, ports.AccionAlta, 0, p.ClaveOperacion, huellas, indice)
	v3, err := autorizar(ctx, orden, m)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	nueva, err := s.fichaNueva(ctx, orden.Identidad, contacto, indice, exig.CatalogoRef)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	recibo, err := s.d.Registro.Alta(ctx, orden, m, v3, nueva)
	if err != nil {
		return ports.ReciboFicha{}, nominal(err)
	}
	if !reciboValido(recibo, ports.AccionAlta, 1) {
		return ports.ReciboFicha{}, ports.ErrNoDisponible
	}
	return recibo, nil
}

func (s *ServicioFichaPropia) fichaNueva(ctx context.Context, identidad domain.IdentidadAcreditada, contacto map[domain.CampoFicha]string, indice ports.IndiceDocumento, catalogo string) (ports.FichaNueva, error) {
	asp, err := domain.GenerarReferenciaAspirante(s.d.Azar)
	if err != nil {
		return ports.FichaNueva{}, ports.ErrNoDisponible
	}
	docRef, err := domain.GenerarReferenciaDocumento(s.d.Azar)
	if err != nil {
		return ports.FichaNueva{}, ports.ErrNoDisponible
	}
	doc := identidad.Documento()
	claro := []byte(doc.Numero)
	sobreDoc, err := s.d.Protector.CifrarDocumento(ctx, asp, docRef, claro)
	borrar(claro)
	if err != nil || !sobreValido(sobreDoc) {
		return ports.FichaNueva{}, ports.ErrNoDisponible
	}
	valores := identidad.Valores()
	for c, v := range contacto {
		valores[c] = v
	}
	cifrados := make([]ports.ValorCifrado, 0, len(valores))
	for _, campo := range domain.CamposOrdenados(valores) {
		claro := []byte(valores[campo])
		sobre, err := s.d.Protector.CifrarValor(ctx, asp, campo, 1, claro)
		borrar(claro)
		if err != nil || !sobreValido(sobre) {
			return ports.FichaNueva{}, ports.ErrNoDisponible
		}
		cifrados = append(cifrados, ports.ValorCifrado{Campo: campo, Version: 1, Origen: campo.OrigenEsperado(), Sobre: &sobre})
	}
	return ports.FichaNueva{
		AspiranteRef: asp,
		Documento:    ports.DocumentoCifrado{DocumentoRef: docRef, Tipo: doc.Tipo, Pais: doc.Pais, Sobre: sobreDoc, Indice: indice},
		Valores:      cifrados, CatalogoRef: catalogo,
	}, nil
}

func sobreValido(s ports.SobreCifrado) bool {
	return s.ClaveRef != "" && len(s.Nonce) == 12 && len(s.Cifrado) > 16
}

// Rectificar cambia, añade o retira datos de contacto con un motivo. El
// registro revela, ya bloqueada la ficha, qué campos tenían valor; el motivo
// se coteja con eso antes de cifrar.
func (s *ServicioFichaPropia) Rectificar(ctx context.Context, orden ports.OrdenFicha, p PeticionFicha) (ports.ReciboFicha, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	motivo := domain.MotivoCambio(p.Motivo)
	campos, err := camposPeticion(p)
	if err != nil || !claveValida(p.ClaveOperacion) || p.VersionEsperada == 0 || p.VersionEsperada >= math.MaxInt64 || !motivo.ValidoParaRectificar() || len(campos) == 0 {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	exig, err := s.exigencias(ctx)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	cambios, err := domain.ContactoPedido(campos, exig.Campos, false)
	if err != nil {
		return ports.ReciboFicha{}, ports.ErrInvalida
	}
	indice, err := s.indice(ctx, orden)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	pre, err := preimagen(actor.PersonaRef, ports.AccionRectificar, p.VersionEsperada, motivo, exig.CatalogoRef, indice, cambios)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	huellas, err := s.sellar(ctx, pre)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	m := material(actor, ports.AccionRectificar, p.VersionEsperada, p.ClaveOperacion, huellas, indice)
	v3, err := autorizar(ctx, orden, m)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	cifrador := &cifradorCambios{protector: s.d.Protector, cambios: cambios, motivo: motivo, version: p.VersionEsperada}
	recibo, err := s.d.Registro.Rectificar(ctx, orden, m, v3, motivo, exig.CatalogoRef, cifrador)
	if err != nil {
		return ports.ReciboFicha{}, nominal(err)
	}
	if !reciboValido(recibo, ports.AccionRectificar, p.VersionEsperada+1) {
		return ports.ReciboFicha{}, ports.ErrNoDisponible
	}
	return recibo, nil
}

type cifradorCambios struct {
	protector ports.ProtectorFicha
	cambios   map[domain.CampoFicha]string
	motivo    domain.MotivoCambio
	version   uint64
}

func (c *cifradorCambios) CifrarCambios(ctx context.Context, estado ports.EstadoParaCambio) ([]ports.ValorCifrado, error) {
	if c == nil || c.protector == nil || !domain.ReferenciaAspiranteValida(estado.AspiranteRef) {
		return nil, ports.ErrNoDisponible
	}
	if estado.Version != c.version {
		return nil, ports.ErrConflicto
	}
	if domain.ValidarMotivo(c.motivo, c.cambios, estado.Presentes) != nil {
		return nil, ports.ErrInvalida
	}
	nueva := c.version + 1
	r := make([]ports.ValorCifrado, 0, len(c.cambios))
	for _, campo := range domain.CamposOrdenados(c.cambios) {
		v := ports.ValorCifrado{Campo: campo, Version: nueva, Origen: domain.OrigenTitular}
		if valor := c.cambios[campo]; valor != "" {
			claro := []byte(valor)
			sobre, err := c.protector.CifrarValor(ctx, estado.AspiranteRef, campo, nueva, claro)
			borrar(claro)
			if err != nil || !sobreValido(sobre) {
				return nil, ports.ErrNoDisponible
			}
			v.Sobre = &sobre
		}
		r = append(r, v)
	}
	return r, nil
}
