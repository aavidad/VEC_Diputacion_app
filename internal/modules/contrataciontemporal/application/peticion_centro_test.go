package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadPeticionCentroPrueba struct {
	actor  domain.ActorPeticionCentro
	config domain.ConfiguracionPeticionCentro
}

type autoridadPuestoPeticionCentroPrueba struct {
	*autoridadPeticionCentroPrueba
	cotejos int
}

func (a *autoridadPuestoPeticionCentroPrueba) ValidarPuestoRPTPeticionCentro(_ context.Context,
	_ domain.ActorPeticionCentro, _ domain.SolicitudCentro, puesto ports.PuestoRPTAltaVerificado,
) error {
	a.cotejos++
	if puesto.PuestoCodigo != "3388" || !puesto.ExisteEnPublicacion {
		return domain.ErrPeticionCentroInvalida
	}
	return nil
}

func TestPeticionCentroV3SellaRatificaYRepiteSinNuevaPublicacion(t *testing.T) {
	_, autoridadBase, repo, solicitud, ahora := servicioPeticionCentroPrueba(t)
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	registro, _ := construirServicioRegistro(t, escenario)
	if err := registro.ConfigurarFuenteNecesidadesAlta(fuente); err != nil {
		t.Fatal(err)
	}
	verificador := &verificadorPuestoRPTPrueba{existe: true}
	if err := registro.ConfigurarVerificadorPuestoRPTAlta(verificador); err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadPuestoPeticionCentroPrueba{autoridadPeticionCentroPrueba: autoridadBase}
	servicio, err := NuevoServicioPeticionCentro(autoridad, repo, relojPeticionCentroPrueba{ahora})
	if err != nil || servicio.ConfigurarPreparadorNecesidad(registro) != nil {
		t.Fatal("preparador de alta existente no compuesto", err)
	}
	solicitud.MotivoClave = "vacante"
	solicitud.Periodo = escenario.solicitud.NecesidadEntrada.Periodo
	entrada := *escenario.solicitud.NecesidadEntrada
	entrada.Campos = make(map[string]string, len(escenario.solicitud.NecesidadEntrada.Campos))
	for clave, valor := range escenario.solicitud.NecesidadEntrada.Campos {
		entrada.Campos[clave] = valor
	}
	entrada.Campos["numero_personas"] = "1"
	comando := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc",
		EsquemaAlta:       ports.EsquemaAltaNecesidadV1, NecesidadEntrada: &entrada, Solicitud: solicitud}
	primero, err := servicio.Ejecutar(t.Context(), comando)
	if err != nil || repo.confirmadas != 1 || repo.ultimo.Peticion.Solicitud.Necesidad == nil ||
		repo.ultimo.Peticion.Solicitud.Necesidad.ValidarInstantanea() != nil ||
		fuente.resoluciones != 1 || verificador.llamadas != 1 || autoridad.cotejos != 1 {
		t.Fatalf("presentación v3: recibo=%+v error=%v fuente=%d rpt=%d cotejos=%d", primero, err,
			fuente.resoluciones, verificador.llamadas, autoridad.cotejos)
	}
	necesidadOriginal, _ := repo.ultimo.Peticion.Solicitud.Necesidad.Clonar()
	fuente.catalogo = domain.CatalogoNecesidadesAlta{} // Publicación posterior retirada.
	repetido, err := servicio.Ejecutar(t.Context(), comando)
	if err != nil || repetido.ReciboRef != primero.ReciboRef || fuente.resoluciones != 1 ||
		verificador.llamadas != 1 || autoridad.cotejos != 1 ||
		!reflect.DeepEqual(repo.ultimo.Peticion.Solicitud.Necesidad, &necesidadOriginal) {
		t.Fatalf("replay modificó publicación original: recibo=%+v error=%v", repetido, err)
	}
	cruzado, err := comando.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	cruzado.NecesidadEntrada.Campos["puesto_codigo"] = "otro"
	if _, err := servicio.Ejecutar(t.Context(), cruzado); !errors.Is(err, ports.ErrClavePeticionCentroUsada) {
		t.Fatalf("misma clave con puesto alterado: %v", err)
	}
	autoridadBase.actor = autoridadBase.config.Ratificador
	ratificar := ports.ComandoPeticionCentro{Operacion: ports.OperacionRatificarPeticionCentro,
		ClaveIdempotencia: "87654321-4321-4321-8321-cba987654321", PeticionRef: primero.PeticionRef,
		VersionEsperada: 1, Motivo: "Revisada por segunda identidad"}
	if _, err := servicio.Ejecutar(t.Context(), ratificar); err != nil ||
		repo.ultimo.Peticion.Estado != "ratificada" ||
		!reflect.DeepEqual(repo.ultimo.Peticion.Solicitud.Necesidad, &necesidadOriginal) ||
		fuente.resoluciones != 1 {
		t.Fatalf("ratificación alteró necesidad publicada: %v", err)
	}
}

func TestPeticionCentroFuenteV3ExigeEsquemaSoloAlCrearYConservaReplayV2(t *testing.T) {
	servicio, _, repo, solicitud, _ := servicioPeticionCentroPrueba(t)
	legado := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc", Solicitud: solicitud}
	primero, err := servicio.Ejecutar(t.Context(), legado)
	if err != nil {
		t.Fatal(err)
	}
	escenario, fuente := escenarioNecesidadRegistroPrueba(t)
	registro, _ := construirServicioRegistro(t, escenario)
	if err := registro.ConfigurarFuenteNecesidadesAlta(fuente); err != nil ||
		servicio.ConfigurarPreparadorNecesidad(registro) != nil {
		t.Fatal("fuente v3 no configurada", err)
	}
	repetido, err := servicio.Ejecutar(t.Context(), legado)
	if err != nil || repetido.ReciboRef != primero.ReciboRef || fuente.resoluciones != 0 || repo.confirmadas != 2 {
		t.Fatalf("replay legado alterado: recibo=%+v err=%v", repetido, err)
	}
	nuevo := legado
	nuevo.ClaveIdempotencia = "87654321-4321-4321-8321-cba987654321"
	if _, err := servicio.Ejecutar(t.Context(), nuevo); !errors.Is(err, domain.ErrPeticionCentroInvalida) ||
		repo.confirmadas != 2 || fuente.resoluciones != 0 {
		t.Fatalf("nueva petición v2 atravesó fuente v3: %v", err)
	}
}

func (a *autoridadPeticionCentroPrueba) ActorPeticionCentro(context.Context) (domain.ActorPeticionCentro, error) {
	return a.actor, nil
}
func (a *autoridadPeticionCentroPrueba) ConfiguracionPeticionCentro(context.Context, domain.ActorPeticionCentro) (domain.ConfiguracionPeticionCentro, error) {
	return a.config, nil
}

type repositorioPeticionCentroPrueba struct {
	operaciones map[string]ports.MaterialPeticionCentro
	peticiones  map[string]domain.DatosPeticionCentro
	confirmadas int
	ultimo      ports.MaterialPeticionCentro
	reciboMalo  bool
}

func (r *repositorioPeticionCentroPrueba) ConsultarOperacion(_ context.Context, _ domain.ActorPeticionCentro, clave string) (*ports.MaterialPeticionCentro, error) {
	m, existe := r.operaciones[clave]
	if !existe {
		return nil, nil
	}
	return &m, nil
}
func (r *repositorioPeticionCentroPrueba) ObtenerPeticion(_ context.Context, _ domain.ActorPeticionCentro, ref string) (domain.DatosPeticionCentro, error) {
	return r.peticiones[ref], nil
}
func (r *repositorioPeticionCentroPrueba) ConfirmarPeticion(_ context.Context, m ports.MaterialPeticionCentro) (ports.ReciboPeticionCentro, error) {
	r.confirmadas++
	r.ultimo = m
	r.operaciones[m.Comando.ClaveIdempotencia] = m
	r.peticiones[m.Peticion.Referencia] = m.Peticion
	if r.reciboMalo {
		return ports.ReciboPeticionCentro{}, nil
	}
	registrado := m.Peticion.CreadaEn.Add(time.Minute)
	if m.Peticion.RatificadaEn.After(registrado) {
		registrado = m.Peticion.RatificadaEn
	}
	estado := "registrado"
	if r.confirmadas > 1 {
		estado = "replay_confirmado"
	}
	return ports.ReciboPeticionCentro{ReciboRef: "recibo:peticion:001", PeticionRef: m.Peticion.Referencia,
		Version: m.Peticion.Version, Estado: m.Peticion.Estado, ActorRef: m.Actor.ActorRef,
		RegistradoEn: registrado, EstadoLocal: estado}, nil
}

type relojPeticionCentroPrueba struct{ ahora time.Time }

func (r relojPeticionCentroPrueba) Ahora() time.Time { return r.ahora }

func TestPeticionCentroExigeJornadaPersonasYPuestoAntesDeEscribir(t *testing.T) {
	servicio, _, repo, solicitud, _ := servicioPeticionCentroPrueba(t)
	for _, caso := range []struct {
		nombre string
		mudar  func(*domain.SolicitudCentro)
	}{
		{"sin jornada", func(s *domain.SolicitudCentro) { s.JornadaMinutos = 0 }},
		{"sin personas", func(s *domain.SolicitudCentro) { s.NumeroPersonas = 0 }},
		{"sin puesto", func(s *domain.SolicitudCentro) { s.PuestoSolicitado = "" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			entrada := *solicitud
			caso.mudar(&entrada)
			_, err := servicio.Ejecutar(context.Background(), ports.ComandoPeticionCentro{
				Operacion:         ports.OperacionPresentarPeticionCentro,
				ClaveIdempotencia: "f3134ee2-61af-467d-aa58-dc71f07553b6", Solicitud: &entrada,
			})
			if !errors.Is(err, domain.ErrPeticionCentroInvalida) || repo.confirmadas != 0 {
				t.Fatalf("petición incompleta escrita: err=%v confirmadas=%d", err, repo.confirmadas)
			}
		})
	}
}

type preparadorPeriodoCentroPrueba struct {
	llamadas int
	fallar   bool
}

func (p *preparadorPeriodoCentroPrueba) PrepararPeriodoModalidad(_ context.Context, _ domain.ClaveCatalogo, periodo domain.PeriodoPrevisto) (domain.PeriodoPrevisto, error) {
	p.llamadas++
	if p.fallar {
		return domain.PeriodoPrevisto{}, domain.ErrDatoInvalido
	}
	periodo.PoliticaFin = domain.PoliticaFin{
		ReglaRef: "catalogo:3:c12.modalidad.sustitucion", CatalogoVersion: 3,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		FechaFin:             "no_aplica", CausaFin: "reincorporacion_titular",
	}
	return periodo, periodo.Validar()
}

func TestPeticionCentroFinPorCausaReusaSnapshotTrasCambiarRegla(t *testing.T) {
	_, autoridad, repo, solicitud, ahora := servicioPeticionCentroPrueba(t)
	solicitud.Periodo.Fin = time.Time{}
	solicitud.Periodo.CausaFin = "reincorporacion_titular"
	preparador := &preparadorPeriodoCentroPrueba{}
	servicio, err := NuevoServicioPeticionCentro(autoridad, repo, relojPeticionCentroPrueba{ahora}, preparador)
	if err != nil {
		t.Fatal(err)
	}
	comando := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc", Solicitud: solicitud}
	primero, err := servicio.Ejecutar(context.Background(), comando)
	if err != nil || repo.ultimo.Peticion.Solicitud.Periodo.PoliticaFin.CatalogoVersion != 3 || preparador.llamadas != 1 {
		t.Fatalf("snapshot inicial: recibo=%#v err=%v llamadas=%d", primero, err, preparador.llamadas)
	}
	preparador.fallar = true // Simula catálogo cambiado o retirado después del COMMIT.
	replay, err := servicio.Ejecutar(context.Background(), comando)
	if err != nil || replay.ReciboRef != primero.ReciboRef || preparador.llamadas != 1 || repo.confirmadas != 2 {
		t.Fatalf("replay revalidó regla vigente: recibo=%#v err=%v llamadas=%d", replay, err, preparador.llamadas)
	}
}

func TestServicioPeticionCentroPresentaRatificaYReplayConfirma(t *testing.T) {
	servicio, autoridad, repo, solicitud, ahora := servicioPeticionCentroPrueba(t)
	clave := "12345678-1234-4234-8234-123456789abc"
	presentar := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro, ClaveIdempotencia: clave, Solicitud: solicitud}
	recibo, err := servicio.Ejecutar(context.Background(), presentar)
	if err != nil || recibo.Version != 1 || repo.ultimo.Actor != autoridad.config.Solicitante {
		t.Fatalf("presentar: recibo=%#v material=%#v error=%v", recibo, repo.ultimo, err)
	}
	original := repo.ultimo
	replay, err := servicio.Ejecutar(context.Background(), presentar)
	if err != nil || replay.EstadoLocal != "replay_confirmado" || repo.confirmadas != 2 || !reflect.DeepEqual(repo.ultimo, original) {
		t.Fatalf("replay no reconfirmado con material original: %#v, %v", replay, err)
	}
	solicitud.DocumentosAdjuntos[0] = "documento:alterado"
	if original.Peticion.Solicitud.DocumentosAdjuntos[0] != "documento:001" || original.Comando.ClaveIdempotencia != clave {
		t.Fatal("material no conservó clon y UUID originales")
	}
	autoridad.actor = autoridad.config.Ratificador
	servicio.reloj = relojPeticionCentroPrueba{ahora.Add(2 * time.Minute)}
	ratificar := ports.ComandoPeticionCentro{Operacion: ports.OperacionRatificarPeticionCentro,
		ClaveIdempotencia: "abcdefab-cdef-4abc-8def-abcdefabcdef", PeticionRef: recibo.PeticionRef, VersionEsperada: 1, Motivo: "Confirmada"}
	final, err := servicio.Ejecutar(context.Background(), ratificar)
	if err != nil || final.Version != 2 || final.Estado != "ratificada" || repo.ultimo.Actor != autoridad.config.Ratificador {
		t.Fatalf("ratificar: %#v, %v", final, err)
	}
}

func TestServicioPeticionCentroConflictosDenegacionesYFallosNoEscriben(t *testing.T) {
	servicio, autoridad, repo, solicitud, _ := servicioPeticionCentroPrueba(t)
	clave := "12345678-1234-4234-8234-123456789abc"
	base := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro, ClaveIdempotencia: clave, Solicitud: solicitud}
	if _, err := servicio.Ejecutar(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	escritas := repo.confirmadas
	divergente := base
	copia := *solicitud
	copia.Detalle = "Otra necesidad"
	divergente.Solicitud = &copia
	if _, err := servicio.Ejecutar(context.Background(), divergente); !errors.Is(err, ports.ErrClavePeticionCentroUsada) || repo.confirmadas != escritas {
		t.Fatalf("UUID divergente: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
	actorAjeno := autoridad.config.Ratificador
	actorAjeno.PuestoRef = "puesto:otro"
	autoridad.actor = actorAjeno
	ratificar := ports.ComandoPeticionCentro{Operacion: ports.OperacionRatificarPeticionCentro,
		ClaveIdempotencia: "abcdefab-cdef-4abc-8def-abcdefabcdef", PeticionRef: repo.ultimo.Peticion.Referencia, VersionEsperada: 1, Motivo: "Confirmada"}
	if _, err := servicio.Ejecutar(context.Background(), ratificar); !errors.Is(err, domain.ErrRatificacionCentroDenegada) || repo.confirmadas != escritas {
		t.Fatalf("actor ajeno: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
	ratificar.VersionEsperada = 2
	if _, err := servicio.Ejecutar(context.Background(), ratificar); !errors.Is(err, domain.ErrPeticionCentroInvalida) || repo.confirmadas != escritas {
		t.Fatalf("versión inválida: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
	autoridad.actor = autoridad.config.Solicitante
	repo.reciboMalo = true
	base.ClaveIdempotencia = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := servicio.Ejecutar(context.Background(), base); !errors.Is(err, ports.ErrReciboPeticionCentroNoConfiable) {
		t.Fatalf("recibo inválido: %v", err)
	}
	antes := repo.confirmadas
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := servicio.Ejecutar(ctx, base); !errors.Is(err, context.Canceled) || repo.confirmadas != antes {
		t.Fatalf("cancelación confirmó: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
}

func servicioPeticionCentroPrueba(t *testing.T) (*ServicioPeticionCentro, *autoridadPeticionCentroPrueba, *repositorioPeticionCentroPrueba, *domain.SolicitudCentro, time.Time) {
	t.Helper()
	centro := "centro:sintetico:001"
	config := domain.ConfiguracionPeticionCentro{Referencia: "config:peticion:001", Version: 1,
		Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicitante", PerfilRef: "perfil:responsable", CentroRef: centro, PuestoRef: "puesto:jefatura"},
		Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratificador", PerfilRef: "perfil:responsable", CentroRef: centro, PuestoRef: "puesto:direccion"}}
	autoridad := &autoridadPeticionCentroPrueba{actor: config.Solicitante, config: config}
	repo := &repositorioPeticionCentroPrueba{operaciones: map[string]ports.MaterialPeticionCentro{}, peticiones: map[string]domain.DatosPeticionCentro{}}
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	solicitud := &domain.SolicitudCentro{CentroRef: centro, ContactoRef: "contacto:sintetico:001", CategoriaRef: "categoria:tecnica",
		GrupoSubgrupo: "A1", MotivoClave: "necesidad.temporal", Detalle: "Necesidad sintética",
		Periodo: domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.Add(24 * time.Hour)}, DocumentosAdjuntos: []string{"documento:001"},
		JornadaMinutos: 2250, NumeroPersonas: 1, PuestoSolicitado: "Técnico de administración"}
	servicio, err := NuevoServicioPeticionCentro(autoridad, repo, relojPeticionCentroPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return servicio, autoridad, repo, solicitud, ahora
}
