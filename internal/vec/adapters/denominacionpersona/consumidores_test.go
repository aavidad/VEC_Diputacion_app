package denominacionpersona

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type personasPrueba struct {
	ausente  bool
	llamadas int
}

func (p *personasPrueba) RevalidarPersonaDenominacion(context.Context, string) error {
	p.llamadas++
	if p.ausente {
		return errors.New("persona_ausente")
	}
	return nil
}

type fuenteAutorizadaPrueba struct {
	falloAcuse            bool
	antesRevalidar        func()
	sobre                 ports.SobreDenominacionPersona
	fallaCommit, revocada bool
	consumida, revalidada int
	indice                ports.IndiceDenominacionPersona
}

func (f *fuenteAutorizadaPrueba) LeerDenominacionPersonaAutorizada(_ context.Context, a ports.AccesoDenominacionPersona) (ports.LecturaDenominacionPersonaConfirmada, error) {
	if f.fallaCommit {
		return ports.LecturaDenominacionPersonaConfirmada{}, errors.New("commit_no_confirmado")
	}
	f.consumida++
	acuse, _ := domain.NuevoAcuseConsumoDenominacionPersona(domain.DatosAcuseConsumoDenominacionPersona{ConsumoRef: "consumo:prueba", AuditoriaRef: "auditoria:prueba", DecisionRef: "decision:prueba", CorrelacionRef: a.Auditoria.CorrelationRef, RecursoRef: a.Recurso.Referencia, PersonaRef: a.PersonaRef, Version: a.Version, SobreSHA256: huellaSobre(f.sobre)})
	return ports.LecturaDenominacionPersonaConfirmada{Sobre: f.sobre, Acuse: acuse}, nil
}
func (f *fuenteAutorizadaPrueba) ValidarAcuseDenominacionPersona(context.Context, ports.AccesoDenominacionPersona, ports.LecturaDenominacionPersonaConfirmada) error {
	if f.falloAcuse {
		return errors.New("acuse_invalido")
	}
	return nil
}
func (f *fuenteAutorizadaPrueba) RevalidarAccesoDenominacionPersona(context.Context, ports.AccesoDenominacionPersona, ports.SobreDenominacionPersona) error {
	f.revalidada++
	if f.antesRevalidar != nil {
		f.antesRevalidar()
	}
	if f.revocada {
		return errors.New("rol_revocado")
	}
	return nil
}
func (f *fuenteAutorizadaPrueba) BuscarDenominacionPersonaAutorizada(_ context.Context, _ ports.AccesoDenominacionPersona, i ports.IndiceDenominacionPersona, _ int) ([]ports.ReferenciaDenominacionPersona, error) {
	f.indice = i
	if f.fallaCommit {
		return nil, errors.New("commit_no_confirmado")
	}
	return []ports.ReferenciaDenominacionPersona{{PersonaRef: personaUno, Version: 1}, {PersonaRef: personaDos, Version: 1}}, nil
}

func TestPreparacionRevalidaPersonaCanonica(t *testing.T) {
	p, _ := escenario(t)
	personas := &personasPrueba{}
	preparador, err := NuevoPreparador(p, personas)
	if err != nil {
		t.Fatal("constructor")
	}
	prep, err := preparador.PrepararDenominacionPersona(context.Background(), personaUno, 1, "procedencia:declarada", ambitoPrueba, "kms:denom:busqueda:1", []byte("Nombre Sintético"))
	if err != nil || personas.llamadas != 2 || prep.VersionEsperada != 1 || prep.Sobre.Version != 2 {
		t.Fatal("preparacion_versionada")
	}
	personas.ausente = true
	if _, err := preparador.PrepararDenominacionPersona(context.Background(), personaUno, 0, "procedencia:declarada", ambitoPrueba, "kms:denom:busqueda:1", []byte("Nombre Sintético")); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("prefijo_no_acredita_persona")
	}
}

func TestLecturaConfirmaAntesDelCallbackYRevalidaTrasKMS(t *testing.T) {
	for _, caso := range []string{"ok", "commit", "rol", "kms", "persona", "referencia", "acuse", "kms_tras_revalidar"} {
		t.Run(caso, func(t *testing.T) {
			p, claves := escenario(t)
			prep := preparar(t, p, personaUno, 0)
			fuente := &fuenteAutorizadaPrueba{sobre: prep.Sobre}
			personas := &personasPrueba{}
			intentos := &intentosPrueba{}
			lector, _ := NuevoLector(p, personas, fuente, intentos, configIntentosPrueba())
			switch caso {
			case "acuse":
				fuente.falloAcuse = true
			case "kms_tras_revalidar":
				fuente.antesRevalidar = func() { claves.claves.Cifrado.Revocada = true }
			case "commit":
				fuente.fallaCommit = true
			case "rol":
				fuente.revocada = true
			case "kms":
				claves.error = true
			case "persona":
				personas.ausente = true
			case "referencia":
				fuente.sobre.PersonaRef = personaDos
			}
			llamado := false
			err := lector.ConDenominacionPersona(context.Background(), accesoPrueba(t), func(d domain.DenominacionPersona) error {
				llamado = true
				if fuente.consumida != 1 || fuente.revalidada != 1 || personas.llamadas != 1 {
					t.Fatal("orden_autorizacion")
				}
				return d.ConNombreMostrar(func([]byte) error { return nil })
			})
			if caso == "ok" {
				if err != nil || !llamado {
					t.Fatal("lectura")
				}
			} else if !errors.Is(err, ErrNoDisponible) || llamado || intentos.llamadas != 1 {
				t.Fatal("fallo_abierto")
			}
		})
	}
}

func TestBusquedaEntregaTokensYConservaDosPersonas(t *testing.T) {
	p, _ := escenario(t)
	fuente := &fuenteAutorizadaPrueba{}
	lector, _ := NuevoLector(p, &personasPrueba{}, fuente, &intentosPrueba{}, configIntentosPrueba())
	refs, err := lector.BuscarDenominacionPersona(context.Background(), accesoPrueba(t), ambitoPrueba, "kms:denom:busqueda:1", []byte("Nombre Sintético"), 10)
	if err != nil || len(refs) != 2 || len(fuente.indice.Tokens) != 2 || fuente.indice.AmbitoRef != ambitoPrueba {
		t.Fatal("busqueda")
	}
	fuente.fallaCommit = true
	if refs, err := lector.BuscarDenominacionPersona(context.Background(), accesoPrueba(t), ambitoPrueba, "kms:denom:busqueda:1", []byte("Nombre Sintético"), 10); !errors.Is(err, ErrNoDisponible) || refs != nil {
		t.Fatal("commit_no_confirmado")
	}
}

// La fuente V3 y el registrador son dobles: prueban la secuencia y el contrato,
// no conceden autorización productiva ni acreditan persistencia.
type intentosPrueba struct {
	llamadas int
	orden    ports.OrdenIntentoAuditoria
	ambiguo  bool
}

func (i *intentosPrueba) AppendIntentoAuditoria(_ context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	i.llamadas++
	if i.ambiguo && i.llamadas == 1 {
		i.orden = o
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	d, err := o.Datos()
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	if i.ambiguo {
		anterior, _ := i.orden.Datos()
		if anterior.IntentoRef != d.IntentoRef {
			return ports.AcuseIntentoAuditoria{}, errors.New("intento_distinto")
		}
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)}, nil
}
func configIntentosPrueba() ConfiguracionIntentos {
	return ConfiguracionIntentos{Proceso: "vec-server", Canal: "interna_corporativa", MotivoError: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "error_observado"}}
}
func accesoPrueba(t *testing.T) ports.AccesoDenominacionPersona {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), personaUno, "prf_aaaaaaaaaaaaaaaaaaaaaaaa", domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal("evidencia_sintetica")
	}
	return ports.AccesoDenominacionPersona{PersonaRef: personaUno, Version: 1, Contexto: resultado.Contexto, ResultadoContexto: resultado, Vinculo: vinculo, FinalidadRef: "presentacion_persona", Recurso: domain.RecursoAutorizable{ModuloID: "vec", Referencia: "recurso:prueba"}, Auditoria: domain.AuditEntry{CorrelationRef: "correlacion_11111111111111111111111111111111"}}
}
func TestErrorObservadoReutilizaIntentoYExigeEvidenciaOriginal(t *testing.T) {
	p, _ := escenario(t)
	fuente := &fuenteAutorizadaPrueba{sobre: preparar(t, p, personaUno, 0).Sobre, revocada: true}
	intentos := &intentosPrueba{ambiguo: true}
	lector, _ := NuevoLector(p, &personasPrueba{}, fuente, intentos, configIntentosPrueba())
	llamado := false
	if err := lector.ConDenominacionPersona(context.Background(), accesoPrueba(t), func(domain.DenominacionPersona) error { llamado = true; return nil }); !errors.Is(err, ErrNoDisponible) || llamado || intentos.llamadas != 2 || fuente.consumida != 1 {
		t.Fatal("error_observado")
	}
	a := accesoPrueba(t)
	a.ResultadoContexto = domain.ResultadoContextoActorRegistradoV2{}
	if lector.ConDenominacionPersona(context.Background(), a, func(domain.DenominacionPersona) error { return nil }) == nil || fuente.consumida != 1 {
		t.Fatal("evidencia_ausente")
	}
	if _, err := NuevoLector(p, &personasPrueba{}, fuente, nil, configIntentosPrueba()); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("auditoria_ausente")
	}
}
