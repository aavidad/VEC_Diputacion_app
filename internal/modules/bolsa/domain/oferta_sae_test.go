package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func catalogoSAEPrueba() CatalogoOfertaSAE {
	return CatalogoOfertaSAE{
		Referencia: "catalogo:sae:v1", Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		Modalidades: []string{"interinidad"}, CriteriosValoracion: []string{"experiencia"},
		ResultadosValoracion: []string{"apto", "no_apto"}, ResultadosEntrevista: []string{"apto", "no_apto"},
		Plantillas: map[string]string{DocumentoSAESolicitud: "plantilla:solicitud:v1", DocumentoSAEActa: "plantilla:acta:v1", DocumentoSAEResolucion: "plantilla:resolucion:v1"},
	}
}

func ofertaSAEPrueba(t *testing.T) OfertaSAE {
	t.Helper()
	o, err := NuevaOfertaSAE("oferta-sae:001", DatosOfertaSAE{
		CategoriaRef: "categoria:operario", PuestoRef: "puesto:auxiliar", NumeroPlazas: 2,
		Modalidad: "interinidad", Duracion: "Hasta cobertura reglamentaria", Requisitos: "Requisitos según catálogo de la oferta",
		ExpedienteCTRef: "expediente:ct:001",
	}, catalogoSAEPrueba())
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func cambioSAEPrueba(o OfertaSAE, accion, clave string) CambioOfertaSAE {
	return CambioOfertaSAE{Accion: accion, Clave: clave, ReciboRef: "recibo:" + clave,
		ActorRef: "persona:rrhh:001", VersionEsperada: o.Version,
		Instante: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
}

func aplicarSAEPrueba(t *testing.T, o OfertaSAE, c CambioOfertaSAE) OfertaSAE {
	t.Helper()
	siguiente, recibo, reutilizado, err := o.Aplicar(c)
	if err != nil || recibo != c.ReciboRef || reutilizado || siguiente.Version != o.Version+1 {
		t.Fatalf("cambio %s: estado=%+v recibo=%q replay=%v error=%v", c.Accion, siguiente, recibo, reutilizado, err)
	}
	return siguiente
}

func TestOfertaSAERecorridoManualExternoYReplay(t *testing.T) {
	o := ofertaSAEPrueba(t)
	if o.Estado != EstadoSAEPreparada || o.Version != 1 || o.Catalogo.Version != 1 {
		t.Fatalf("alta: %+v", o)
	}
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0001")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/001", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	if o.Estado != EstadoSAEEnviada || o.NumeroSAE != envio.NumeroSAE {
		t.Fatalf("envío manual: %+v", o)
	}
	candidato := CandidatoOfertaSAE{Referencia: "candidato:externo:001", NombreProtegidoRef: "dato:nombre:001", DocumentoProtegidoRef: "dato:documento:001", ContactoProtegidoRef: "dato:contacto:001"}
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0001")
	registro.Candidato = &candidato // Sin empleado ni persona previa: es aspirante externo válido.
	o = aplicarSAEPrueba(t, o, registro)
	if o.Candidatos[0].EstadoConciliacion != ConciliacionSAEPendiente || o.Candidatos[0].PersonaRef != "" {
		t.Fatalf("aspirante externo sin vínculo: %+v", o.Candidatos[0])
	}
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-recepcion-0001"))
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEIniciarSeleccion, "clave-seleccion-0001"))
	valoracion := cambioSAEPrueba(o, AccionSAEValorar, "clave-valoracion-0001")
	valoracion.Valoracion = &ValoracionOfertaSAE{CandidatoRef: candidato.Referencia, Orden: 1, EntrevistaResultado: "apto", Criterios: []ResultadoCriterioSAE{{Criterio: "experiencia", Resultado: "apto"}}}
	o = aplicarSAEPrueba(t, o, valoracion)
	resolucion := cambioSAEPrueba(o, AccionSAEResolver, "clave-resolucion-0001")
	resolucion.CandidatoElegidoRef = candidato.Referencia
	if _, _, _, err := o.Aplicar(resolucion); !errors.Is(err, ErrOfertaSAEConciliacionPendiente) {
		t.Fatalf("resolvió sin persona canónica: %v", err)
	}
	acreditacion := AcreditacionPersonaSAE{PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		EvidenciaRef: "evidencia:persona:001", VerificadaEn: resolucion.Instante.Add(-time.Minute), ValidaHasta: resolucion.Instante.Add(time.Hour)}
	conciliar := cambioSAEPrueba(o, AccionSAEConciliarPersona, "clave-conciliar-0001")
	conciliar.CandidatoElegidoRef, conciliar.Acreditacion = candidato.Referencia, &acreditacion
	o = aplicarSAEPrueba(t, o, conciliar)
	previa := o
	resolucion.VersionEsperada, resolucion.Acreditacion = o.Version, &acreditacion
	caducada := acreditacion
	caducada.ValidaHasta = resolucion.Instante
	resolucion.Acreditacion = &caducada
	if _, _, _, err := o.Aplicar(resolucion); !errors.Is(err, ErrOfertaSAEConciliacionPendiente) {
		t.Fatalf("vínculo caducado: %v", err)
	}
	resolucion.Acreditacion = &acreditacion
	o = aplicarSAEPrueba(t, o, resolucion)
	if o.Estado != EstadoSAEResuelta || o.CandidatoSeleccionadoRef != candidato.Referencia || len(o.Actuaciones) != 7 || previa.Estado != EstadoSAEEnSeleccion {
		t.Fatalf("resolución o copia: %+v", o)
	}
	repetida, recibo, reutilizado, err := o.Aplicar(registro)
	if err != nil || !reutilizado || recibo != registro.ReciboRef || repetida.Version != o.Version || len(repetida.Candidatos) != 1 {
		t.Fatalf("replay: %q %v %v", recibo, reutilizado, err)
	}
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:otro", NombreProtegidoRef: "dato:nombre:002", DocumentoProtegidoRef: "dato:documento:002", ContactoProtegidoRef: "dato:contacto:002"}
	if _, _, _, err = o.Aplicar(registro); !errors.Is(err, ErrOfertaSAEClaveReutilizada) {
		t.Fatalf("clave reutilizada: %v", err)
	}
}

func TestOfertaSAEReplayPersonaConEvidenciaRenovada(t *testing.T) {
	o := ofertaSAEPrueba(t)
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-replay-01")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/901", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-registro-replay-01")
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:replay:001", NombreProtegidoRef: "dato:nombre:901", DocumentoProtegidoRef: "dato:documento:901", ContactoProtegidoRef: "dato:contacto:901"}
	o = aplicarSAEPrueba(t, o, registro)
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-recepcion-replay-01"))
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEIniciarSeleccion, "clave-seleccion-replay-01"))
	valoracion := cambioSAEPrueba(o, AccionSAEValorar, "clave-valoracion-replay-01")
	valoracion.Valoracion = &ValoracionOfertaSAE{CandidatoRef: registro.Candidato.Referencia, Orden: 1, EntrevistaResultado: "apto", Criterios: []ResultadoCriterioSAE{{Criterio: "experiencia", Resultado: "apto"}}}
	o = aplicarSAEPrueba(t, o, valoracion)
	acreditacion := AcreditacionPersonaSAE{PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1,
		EvidenciaRef: "evidencia:persona:901", VerificadaEn: valoracion.Instante.Add(-time.Minute), ValidaHasta: valoracion.Instante.Add(time.Hour)}
	renovada := acreditacion
	renovada.EvidenciaRef = "evidencia:persona:902"
	renovada.PersonaVersion = 2
	renovada.VerificadaEn = acreditacion.VerificadaEn.Add(time.Minute)
	renovada.ValidaHasta = acreditacion.ValidaHasta.Add(time.Hour)
	conciliar := cambioSAEPrueba(o, AccionSAEConciliarPersona, "clave-conciliar-replay-01")
	conciliar.CandidatoElegidoRef, conciliar.Acreditacion = registro.Candidato.Referencia, &acreditacion
	o = aplicarSAEPrueba(t, o, conciliar)
	version := o.Version
	conciliar.Acreditacion = &renovada
	if repetida, recibo, replay, err := o.Aplicar(conciliar); err != nil || !replay || recibo != conciliar.ReciboRef || repetida.Version != version {
		t.Fatalf("conciliar renovada: recibo=%q replay=%v error=%v", recibo, replay, err)
	}
	otraPersona := renovada
	otraPersona.PersonaRef = "per_abcdefghijkl0123456789"
	conciliar.Acreditacion = &otraPersona
	if _, _, _, err := o.Aplicar(conciliar); !errors.Is(err, ErrOfertaSAEClaveReutilizada) {
		t.Fatalf("conciliar con otra Persona: %v", err)
	}
	conciliar.Acreditacion = &renovada
	conciliar.CandidatoElegidoRef = "candidato:replay:otro"
	if _, _, _, err := o.Aplicar(conciliar); !errors.Is(err, ErrOfertaSAEClaveReutilizada) {
		t.Fatalf("conciliar con otro candidato: %v", err)
	}
	for _, accion := range []string{AccionSAEConciliarPersona, AccionSAEResolver} {
		nuevo := cambioSAEPrueba(o, accion, "clave-nueva-"+accion)
		nuevo.CandidatoElegidoRef = registro.Candidato.Referencia
		nuevo.Acreditacion = &acreditacion
		nuevo.VersionEsperada--
		if _, _, _, err := o.Aplicar(nuevo); !errors.Is(err, ErrOfertaSAEVersion) {
			t.Fatalf("%s sin versión vigente: %v", accion, err)
		}
		nuevo.VersionEsperada = o.Version
		caducada := acreditacion
		caducada.ValidaHasta = nuevo.Instante
		nuevo.Acreditacion = &caducada
		if _, _, _, err := o.Aplicar(nuevo); !errors.Is(err, ErrOfertaSAEConciliacionPendiente) {
			t.Fatalf("%s sin atestación vigente: %v", accion, err)
		}
	}

	resolver := cambioSAEPrueba(o, AccionSAEResolver, "clave-resolver-replay-01")
	resolver.CandidatoElegidoRef, resolver.Acreditacion = registro.Candidato.Referencia, &renovada
	resolver.Instante = acreditacion.ValidaHasta.Add(time.Minute) // La conciliación histórica ya caducó.
	o = aplicarSAEPrueba(t, o, resolver)
	version = o.Version
	resolver.Acreditacion = &acreditacion
	if repetida, recibo, replay, err := o.Aplicar(resolver); err != nil || !replay || recibo != resolver.ReciboRef || repetida.Version != version {
		t.Fatalf("resolver renovada: recibo=%q replay=%v error=%v", recibo, replay, err)
	}
	resolver.Acreditacion = &otraPersona
	if _, _, _, err := o.Aplicar(resolver); !errors.Is(err, ErrOfertaSAEClaveReutilizada) {
		t.Fatalf("resolver con otra Persona: %v", err)
	}
	resolver.Acreditacion = &renovada
	resolver.CandidatoElegidoRef = "candidato:replay:otro"
	if _, _, _, err := o.Aplicar(resolver); !errors.Is(err, ErrOfertaSAEClaveReutilizada) {
		t.Fatalf("resolver con otro candidato: %v", err)
	}
}

func TestOfertaSAEDeniegaPersonaLibreYDuplicadoLogico(t *testing.T) {
	o := ofertaSAEPrueba(t)
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0101")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/101", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	primer := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0101")
	primer.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:externo:101", PersonaRef: "per_0123456789abcdefghijkl",
		NombreProtegidoRef: "dato:nombre:101", DocumentoProtegidoRef: "dato:documento:101", ContactoProtegidoRef: "dato:contacto:101"}
	if _, _, _, err := o.Aplicar(primer); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("per_ libre aceptado: %v", err)
	}
	primer.Candidato.PersonaRef = ""
	o = aplicarSAEPrueba(t, o, primer)
	segundo := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0102")
	segundo.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:externo:102", NombreProtegidoRef: "dato:nombre:102",
		DocumentoProtegidoRef: "dato:documento:102", ContactoProtegidoRef: "dato:contacto:102"}
	o = aplicarSAEPrueba(t, o, segundo)
	acreditacion := AcreditacionPersonaSAE{PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1,
		EvidenciaRef: "evidencia:persona:101", VerificadaEn: primer.Instante.Add(-time.Minute), ValidaHasta: primer.Instante.Add(time.Hour)}
	conciliar := cambioSAEPrueba(o, AccionSAEConciliarPersona, "clave-conciliar-0101")
	conciliar.CandidatoElegidoRef, conciliar.Acreditacion = primer.Candidato.Referencia, &acreditacion
	o = aplicarSAEPrueba(t, o, conciliar)
	conciliar2 := cambioSAEPrueba(o, AccionSAEConciliarPersona, "clave-conciliar-0102")
	conciliar2.CandidatoElegidoRef = segundo.Candidato.Referencia
	conciliar2.Acreditacion = &acreditacion
	if _, _, _, err := o.Aplicar(conciliar2); !errors.Is(err, ErrOfertaSAEInvalida) {
		t.Fatalf("doble candidatura Persona: %v", err)
	}
}

func TestOfertaSAECierraSiSeleccionManualDeclaraDesierta(t *testing.T) {
	o := ofertaSAEPrueba(t)
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0002")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/002", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0002")
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:externo:002", NombreProtegidoRef: "dato:nombre:002", DocumentoProtegidoRef: "dato:documento:002", ContactoProtegidoRef: "dato:contacto:002"}
	o = aplicarSAEPrueba(t, o, registro)
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-recepcion-0002"))
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEIniciarSeleccion, "clave-seleccion-0002"))
	valoracion := cambioSAEPrueba(o, AccionSAEValorar, "clave-valoracion-0002")
	valoracion.Valoracion = &ValoracionOfertaSAE{CandidatoRef: registro.Candidato.Referencia, Orden: 1, EntrevistaResultado: "no_apto", Criterios: []ResultadoCriterioSAE{{Criterio: "experiencia", Resultado: "no_apto"}}}
	o = aplicarSAEPrueba(t, o, valoracion)
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEDeclararDesierta, "clave-desierta-0002"))
	if o.Estado != EstadoSAEDesierta {
		t.Fatalf("cierre: %s", o.Estado)
	}
}

func TestOfertaSAEDeniegaReglasAusentesYVersionObsoleta(t *testing.T) {
	c := catalogoSAEPrueba()
	c.Modalidades = nil
	if _, err := NuevaOfertaSAE("oferta-sae:001", ofertaSAEPrueba(t).Datos, c); !errors.Is(err, ErrOfertaSAEInvalida) {
		t.Fatalf("catálogo vacío: %v", err)
	}
	o := ofertaSAEPrueba(t)
	if _, _, _, err := o.Aplicar(cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-anticipada-0001")); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("salto de fase: %v", err)
	}
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0003")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/003", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0003")
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:externo:003", NombreProtegidoRef: "dato:nombre:003", DocumentoProtegidoRef: "dato:documento:003", ContactoProtegidoRef: "dato:contacto:003"}
	registro.VersionEsperada--
	if _, _, _, err := o.Aplicar(registro); !errors.Is(err, ErrOfertaSAEVersion) {
		t.Fatalf("versión obsoleta: %v", err)
	}
	registro.VersionEsperada = o.Version
	registro.Candidato.DocumentoProtegidoRef = ""
	if _, _, _, err := o.Aplicar(registro); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("documento sin protección: %v", err)
	}
	envio.NumeroSAE = "SAE/2026/003\nOtra línea"
	if _, _, _, err := ofertaSAEPrueba(t).Aplicar(envio); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("número con salto de línea: %v", err)
	}
}
