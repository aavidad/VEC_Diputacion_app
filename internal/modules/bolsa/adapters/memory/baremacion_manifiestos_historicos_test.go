package memory

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type protectorManifiestosRotablePrueba struct {
	mu           sync.RWMutex
	claveActiva  string
	claves       map[string][]byte
	revocadas    map[string]struct{}
	indisponible atomic.Bool
}

func nuevoProtectorManifiestosRotablePrueba() *protectorManifiestosRotablePrueba {
	return &protectorManifiestosRotablePrueba{
		claveActiva: "manifiesto_v1",
		claves: map[string][]byte{
			"manifiesto_v1": []byte("clave-manifiesto-v1-pruebas-32-bytes"),
			"manifiesto_v2": []byte("clave-manifiesto-v2-pruebas-32-bytes"),
		},
		revocadas: make(map[string]struct{}),
	}
}

func (p *protectorManifiestosRotablePrueba) SellarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudSellarSelloBaremacion,
) (string, error) {
	if ctx == nil || solicitud.Finalidad != puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3 {
		return "", puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if p.indisponible.Load() {
		return "", errors.New("llavero de manifiestos indisponible")
	}
	p.mu.RLock()
	identificador := p.claveActiva
	clave := append([]byte(nil), p.claves[identificador]...)
	_, revocada := p.revocadas[identificador]
	p.mu.RUnlock()
	if len(clave) == 0 || revocada {
		return "", errors.New("clave activa de manifiesto no utilizable")
	}
	return calcularSelloManifiestoRotablePrueba(solicitud, identificador, clave)
}

func (p *protectorManifiestosRotablePrueba) VerificarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	if solicitud.Finalidad != puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3 {
		return (verificadorHMACMemoriaPrueba{}).VerificarSelloBaremacion(ctx, solicitud)
	}
	if ctx == nil || solicitud.Validar() != nil || p.indisponible.Load() {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	partes := strings.Split(solicitud.SelloHMAC, ":")
	if len(partes) != 3 {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	p.mu.RLock()
	clave := append([]byte(nil), p.claves[partes[1]]...)
	_, revocada := p.revocadas[partes[1]]
	p.mu.RUnlock()
	if len(clave) == 0 || revocada {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	material, err := solicitud.MaterialCanonicoHMAC()
	if err != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	bytesMaterial := material.Revelar()
	defer borrarBytesManifiestoRotablePrueba(bytesMaterial)
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(bytesMaterial)
	esperado := "hmac-sha256:" + partes[1] + ":" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(esperado), []byte(solicitud.SelloHMAC)) {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

func calcularSelloManifiestoRotablePrueba(
	solicitud puertosbolsa.SolicitudSellarSelloBaremacion,
	identificador string,
	clave []byte,
) (string, error) {
	material, err := solicitud.MaterialCanonicoHMAC()
	if err != nil {
		return "", err
	}
	bytesMaterial := material.Revelar()
	defer borrarBytesManifiestoRotablePrueba(bytesMaterial)
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(bytesMaterial)
	return "hmac-sha256:" + identificador + ":" + hex.EncodeToString(mac.Sum(nil)), nil
}

func borrarBytesManifiestoRotablePrueba(datos []byte) {
	for indice := range datos {
		datos[indice] = 0
	}
}

func (p *protectorManifiestosRotablePrueba) rotarA(identificador string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.claveActiva = identificador
}

func (p *protectorManifiestosRotablePrueba) desconocer(identificador string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.claves, identificador)
}

func (p *protectorManifiestosRotablePrueba) revocar(identificador string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.revocadas[identificador] = struct{}{}
}

type escenarioTresVersionesManifiestoPrueba struct {
	repositorio *RepositorioBaremaciones
	reloj       *relojMemoriaPrueba
	protector   *protectorManifiestosRotablePrueba
	base        dominiobolsa.BaremacionMerito
	versionDos  puertosbolsa.ResultadoConfirmarCambioBaremacion
	versionTres puertosbolsa.ResultadoConfirmarCambioBaremacion
}

func nuevoEscenarioTresVersionesManifiestoPrueba(t *testing.T) escenarioTresVersionesManifiestoPrueba {
	t.Helper()
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	protector := nuevoProtectorManifiestosRotablePrueba()
	repositorio, err := NuevoRepositorioBaremaciones(
		reloj, protector, PerfilRepositorioBaremacionesSoloPruebas(),
	)
	if err != nil {
		t.Fatal(err)
	}
	base := nuevaBaremacionMemoriaPrueba(t)
	alta := confirmarAltaMemoria(t, repositorio, base)

	instanteDos := instanteMemoriaPrueba.Add(15 * time.Minute)
	actualizadaDos := incorporarDecisionMemoriaPrueba(t, base)
	ultimaDos, _ := actualizadaDos.UltimaDecision()
	reservaDos := reservarDecisionHistoricaPrueba(
		t, repositorio, reloj, alta.Version.Referencia, ultimaDos.Contenido.CorrelacionRef, "v2", instanteDos,
	)
	confirmarDos := solicitudConfirmarDecisionHistoricaPrueba(
		t, reservaDos.Token, alta.Version.Referencia, actualizadaDos, "v2", instanteDos, protector,
	)
	versionDos, err := repositorio.ConfirmarCambio(context.Background(), confirmarDos)
	if err != nil {
		t.Fatalf("confirmar version dos: %v", err)
	}

	protector.rotarA("manifiesto_v2")
	instanteTres := instanteMemoriaPrueba.Add(30 * time.Minute)
	actualizadaTres := incorporarRectificacionHistoricaPrueba(t, versionDos.Version.Agregado, "003", instanteTres)
	ultimaTres, _ := actualizadaTres.UltimaDecision()
	reservaTres := reservarDecisionHistoricaPrueba(
		t, repositorio, reloj, versionDos.Version.Referencia, ultimaTres.Contenido.CorrelacionRef, "v3", instanteTres,
	)
	confirmarTres := solicitudConfirmarDecisionHistoricaPrueba(
		t, reservaTres.Token, versionDos.Version.Referencia, actualizadaTres, "v3", instanteTres, protector,
	)
	versionTres, err := repositorio.ConfirmarCambio(context.Background(), confirmarTres)
	if err != nil {
		t.Fatalf("confirmar version tres tras rotacion: %v", err)
	}
	return escenarioTresVersionesManifiestoPrueba{
		repositorio: repositorio, reloj: reloj, protector: protector, base: base,
		versionDos: versionDos, versionTres: versionTres,
	}
}

func reservarDecisionHistoricaPrueba(
	t *testing.T,
	repositorio *RepositorioBaremaciones,
	reloj *relojMemoriaPrueba,
	version puertosbolsa.ReferenciaVersionBaremacion,
	correlacionRef string,
	sufijo string,
	instante time.Time,
) puertosbolsa.ReservaCambioBaremacion {
	t.Helper()
	reloj.fijar(instante)
	solicitud := solicitudReservarDecisionHistoricaPrueba(version, correlacionRef, sufijo, instante)
	reserva, err := repositorio.ReservarCambio(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("reservar decision %s: %v", sufijo, err)
	}
	return reserva
}

func solicitudReservarDecisionHistoricaPrueba(
	version puertosbolsa.ReferenciaVersionBaremacion,
	correlacionRef string,
	sufijo string,
	instante time.Time,
) puertosbolsa.SolicitudReservarCambioBaremacion {
	contexto := contextoMemoriaPruebaAutorizacion(
		puertosbolsa.AccionReservarDecisionBaremacion, version.BaremacionMeritoRef,
		principalBaremacionMemoriaPrueba, "sujeto-001", "autenticacion-"+sufijo,
		correlacionRef, "autorizacion-reserva-"+sufijo, instante,
	)
	return sellarReservaMemoria(puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contexto, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		ClaveIdempotencia: "decision-baremacion-" + sufijo, BaremacionMeritoRef: version.BaremacionMeritoRef,
		VersionEsperada: &version, HuellaSolicitudHMAC: hmacMemoria("0"),
		SolicitadaEn: instante.Add(-time.Minute), ExpiraEn: instante.Add(5 * time.Minute),
	})
}

func solicitudConfirmarDecisionHistoricaPrueba(
	t *testing.T,
	token puertosbolsa.TokenReservaBaremacion,
	version puertosbolsa.ReferenciaVersionBaremacion,
	baremacion dominiobolsa.BaremacionMerito,
	sufijo string,
	instante time.Time,
	sellador puertosbolsa.SelladorSellosBaremacion,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	clon, err := baremacion.ClonarCanonica()
	if err != nil || len(clon.Decisiones) == 0 {
		t.Fatalf("clonar agregado %s: %v", sufijo, err)
	}
	baremacion = clon
	ultima := &baremacion.Decisiones[len(baremacion.Decisiones)-1]
	contexto := contextoMemoriaPruebaAutorizacion(
		puertosbolsa.AccionConfirmarDecisionBaremacion, baremacion.ID,
		principalBaremacionMemoriaPrueba, baremacion.SujetoRef, "autenticacion-"+sufijo,
		ultima.Contenido.CorrelacionRef, "autorizacion-confirmacion-"+sufijo, instante,
	)
	contextoPrevalidacion := contextoMemoriaPruebaAutorizacion(
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion, baremacion.ID,
		principalBaremacionMemoriaPrueba, baremacion.SujetoRef, "autenticacion-"+sufijo,
		ultima.Contenido.CorrelacionRef, "autorizacion-prevalidacion-"+sufijo, instante,
	)
	manifiestoBase, err := manifiestoMemoriaPrueba(
		version, ultima.Contenido, ultima.Firma, contextoPrevalidacion.Proyeccion().AutorizacionRef,
		contexto.Proyeccion().AutorizacionRef, instante,
	)
	if err != nil {
		t.Fatalf("crear manifiesto %s: %v", sufijo, err)
	}
	preparado, representacion, err := manifiestoBase.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto %s: %v", sufijo, err)
	}
	sello, err := sellador.SellarSelloBaremacion(context.Background(), puertosbolsa.SolicitudSellarSelloBaremacion{
		Finalidad: puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, RepresentacionCanonica: representacion,
	})
	if err != nil {
		t.Fatalf("sellar manifiesto %s: %v", sufijo, err)
	}
	manifiesto, err := preparado.IncorporarSello(sello)
	if err != nil {
		t.Fatalf("incorporar sello %s: %v", sufijo, err)
	}
	firma := ultima.Firma
	firma.ManifiestoProbatorioRef = manifiesto.Referencia
	firma.HuellaManifiestoProbatorioSHA256 = manifiesto.HuellaManifiestoSHA256
	firma.SelloManifiestoProbatorioHMACSHA256 = manifiesto.SelloManifiestoHMACSHA256
	decision, err := dominiobolsa.ConstituirDecisionFirmada(ultima.Contenido, firma)
	if err != nil {
		t.Fatalf("reconstituir decision %s: %v", sufijo, err)
	}
	*ultima = decision
	solicitud := puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contexto, ContextoPrevalidacionArchivo: contextoPrevalidacion,
		Token: token, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		VersionEsperada: &version, HuellaSolicitudHMAC: hmacMemoria("0"), Agregado: baremacion,
		Manifiesto: &manifiesto,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "decision_tecnica_firmada", Motivo: "Incorporacion de decision historica " + sufijo + ".",
		},
		ConfirmadaEn: instante,
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("confirmacion historica %s invalida antes del sellado: %v (agregado=%v manifiesto=%v)",
			sufijo, err, solicitud.Agregado.Validar(), solicitud.Manifiesto.Validar())
	}
	return sellarConfirmacionMemoria(solicitud)
}

func incorporarRectificacionHistoricaPrueba(
	t *testing.T,
	baremacion dominiobolsa.BaremacionMerito,
	sufijo string,
	instanteConfirmacion time.Time,
) dominiobolsa.BaremacionMerito {
	t.Helper()
	ultima, existe := baremacion.UltimaDecision()
	if !existe {
		t.Fatal("rectificacion sin decision previa")
	}
	contenidoAnterior := ultima.Contenido
	propuesta := dominiobolsa.PropuestaDecisionTecnica{
		ID: "decision-" + sufijo, CalculoOficial: contenidoAnterior.CalculoOficial,
		PuntosReconocidos: contenidoAnterior.PuntosReconocidos, Resultado: contenidoAnterior.Resultado,
		DecisorRef: contenidoAnterior.DecisorRef, PerfilDecisorClave: contenidoAnterior.PerfilDecisorClave,
		ValoracionesEvidencia: append([]dominiobolsa.ValoracionEvidencia(nil), contenidoAnterior.ValoracionesEvidencia...),
		MotivoClave:           "rectificacion_historica", Motivo: "Rectificacion tecnica trazable " + sufijo + ".",
		FuentesNormativasRefs: append([]string(nil), contenidoAnterior.FuentesNormativasRefs...),
		AutorizacionRef:       "autorizacion-adopcion-" + sufijo, FinalidadClave: contenidoAnterior.FinalidadClave,
		CorrelacionRef: "correlacion-adopcion-" + sufijo, DecididaEn: instanteConfirmacion.Add(-10 * time.Minute),
	}
	contenido, err := baremacion.PrepararRectificacion(propuesta)
	if err != nil {
		t.Fatalf("preparar rectificacion %s: %v", sufijo, err)
	}
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	firma := firmaMemoriaPrueba(contenido, huella, instanteConfirmacion.Add(-9*time.Minute))
	decision, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma)
	if err != nil {
		t.Fatalf("firmar rectificacion %s: %v", sufijo, err)
	}
	actualizada, err := baremacion.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar rectificacion %s: %v", sufijo, err)
	}
	return actualizada
}

func prepararCuartaVersionManifiestoPrueba(
	t *testing.T,
	escenario escenarioTresVersionesManifiestoPrueba,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	instante := instanteMemoriaPrueba.Add(45 * time.Minute)
	agregado := incorporarRectificacionHistoricaPrueba(t, escenario.versionTres.Version.Agregado, "004", instante)
	ultima, _ := agregado.UltimaDecision()
	reserva := reservarDecisionHistoricaPrueba(
		t, escenario.repositorio, escenario.reloj, escenario.versionTres.Version.Referencia,
		ultima.Contenido.CorrelacionRef, "v4", instante,
	)
	return solicitudConfirmarDecisionHistoricaPrueba(
		t, reserva.Token, escenario.versionTres.Version.Referencia, agregado, "v4", instante, escenario.protector,
	)
}

func TestRepositorioBaremacionesConservaManifiestosTrasRotarClave(t *testing.T) {
	escenario := nuevoEscenarioTresVersionesManifiestoPrueba(t)
	for _, version := range []puertosbolsa.ResultadoConfirmarCambioBaremacion{
		escenario.versionDos, escenario.versionTres,
	} {
		_, err := escenario.repositorio.ObtenerVersion(
			context.Background(), puertosbolsa.SolicitudObtenerVersionBaremacion{
				Contexto: contextoMemoriaPrueba(
					puertosbolsa.AccionConsultarVersionBaremacion, escenario.base.ID, escenario.reloj.Ahora(),
				),
				BaremacionMeritoRef: escenario.base.ID, Numero: version.Version.Referencia.Numero,
			},
		)
		if err != nil {
			t.Fatalf("leer version %d tras rotacion: %v", version.Version.Referencia.Numero, err)
		}
	}
	escenario.repositorio.mu.RLock()
	defer escenario.repositorio.mu.RUnlock()
	if len(escenario.repositorio.manifiestosPorReferencia) != 2 ||
		len(escenario.repositorio.manifiestoRefPorVersion) != 2 {
		t.Fatalf("archivo historico incompleto: manifiestos=%d indices=%d",
			len(escenario.repositorio.manifiestosPorReferencia), len(escenario.repositorio.manifiestoRefPorVersion))
	}
}

func TestRepositorioBaremacionesReservaConfirmadaNoRevelaVersionSinClaveHistorica(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*protectorManifiestosRotablePrueba)
	}{
		{
			nombre:  "clave historica revocada",
			alterar: func(p *protectorManifiestosRotablePrueba) { p.revocar("manifiesto_v1") },
		},
		{
			nombre:  "llavero indisponible",
			alterar: func(p *protectorManifiestosRotablePrueba) { p.indisponible.Store(true) },
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
			protector := nuevoProtectorManifiestosRotablePrueba()
			repositorio, err := NuevoRepositorioBaremaciones(
				reloj, protector, PerfilRepositorioBaremacionesSoloPruebas(),
			)
			if err != nil {
				t.Fatal(err)
			}
			base := nuevaBaremacionMemoriaPrueba(t)
			alta := confirmarAltaMemoria(t, repositorio, base)
			instante := instanteMemoriaPrueba.Add(15 * time.Minute)
			agregado := incorporarDecisionMemoriaPrueba(t, base)
			ultima, _ := agregado.UltimaDecision()
			solicitudReserva := solicitudReservarDecisionHistoricaPrueba(
				alta.Version.Referencia, ultima.Contenido.CorrelacionRef, "v2", instante,
			)
			reloj.fijar(instante)
			reserva, err := repositorio.ReservarCambio(context.Background(), solicitudReserva)
			if err != nil {
				t.Fatal(err)
			}
			confirmacion := solicitudConfirmarDecisionHistoricaPrueba(
				t, reserva.Token, alta.Version.Referencia, agregado, "v2", instante, protector,
			)
			confirmada, err := repositorio.ConfirmarCambio(context.Background(), confirmacion)
			if err != nil {
				t.Fatal(err)
			}

			valida, err := repositorio.ReservarCambio(context.Background(), solicitudReserva)
			if err != nil || valida.VersionConfirmada == nil ||
				valida.VersionConfirmada.Referencia != confirmada.Version.Referencia {
				t.Fatalf("retry valido no recupero la version: respuesta=%+v err=%v", valida, err)
			}
			caso.alterar(protector)
			respuesta, err := repositorio.ReservarCambio(context.Background(), solicitudReserva)
			if !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) ||
				!reflect.DeepEqual(respuesta, puertosbolsa.ReservaCambioBaremacion{}) {
				t.Fatalf("se revelo version sin clave historica: respuesta=%+v err=%v", respuesta, err)
			}
			comprobarLongitudesInternas(t, repositorio, 2, 2, 2)
		})
	}
}

func TestRepositorioBaremacionesVerificaTodoElHistoricoAntesDeNuevaVersion(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*escenarioTresVersionesManifiestoPrueba)
	}{
		{"clave antigua desconocida", func(e *escenarioTresVersionesManifiestoPrueba) { e.protector.desconocer("manifiesto_v1") }},
		{"clave antigua revocada", func(e *escenarioTresVersionesManifiestoPrueba) { e.protector.revocar("manifiesto_v1") }},
		{"llavero indisponible", func(e *escenarioTresVersionesManifiestoPrueba) { e.protector.indisponible.Store(true) }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioTresVersionesManifiestoPrueba(t)
			confirmacion := prepararCuartaVersionManifiestoPrueba(t, escenario)
			caso.alterar(&escenario)
			if _, err := escenario.repositorio.ConfirmarCambio(
				context.Background(), confirmacion,
			); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) &&
				!errors.Is(err, puertosbolsa.ErrSelloBaremacionNoAutentico) {
				t.Fatalf("historico no verificable admitido: %v", err)
			}
			comprobarLongitudesInternas(t, escenario.repositorio, 3, 3, 3)
		})
	}
}

func TestRepositorioBaremacionesRechazaHuecosExtrasYSwapsAntesDeNuevaVersion(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*RepositorioBaremaciones)
	}{
		{"hueco de indice", func(r *RepositorioBaremaciones) {
			delete(r.manifiestoRefPorVersion, claveVersionManifiesto("baremacion-001", 2))
		}},
		{"cardinalidad sin manifiesto", func(r *RepositorioBaremaciones) {
			referencia := r.manifiestoRefPorVersion[claveVersionManifiesto("baremacion-001", 2)]
			delete(r.manifiestosPorReferencia, referencia)
		}},
		{"manifiesto extra", func(r *RepositorioBaremaciones) {
			for _, persistido := range r.manifiestosPorReferencia {
				extra := persistido.clonar()
				extra.Manifiesto.Referencia = "manifiesto-probatorio-extra"
				r.manifiestosPorReferencia[extra.Manifiesto.Referencia] = extra
				break
			}
		}},
		{"swap de indices", func(r *RepositorioBaremaciones) {
			claveDos := claveVersionManifiesto("baremacion-001", 2)
			claveTres := claveVersionManifiesto("baremacion-001", 3)
			r.manifiestoRefPorVersion[claveDos], r.manifiestoRefPorVersion[claveTres] =
				r.manifiestoRefPorVersion[claveTres], r.manifiestoRefPorVersion[claveDos]
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioTresVersionesManifiestoPrueba(t)
			confirmacion := prepararCuartaVersionManifiestoPrueba(t, escenario)
			escenario.repositorio.mu.Lock()
			caso.alterar(escenario.repositorio)
			escenario.repositorio.mu.Unlock()
			if _, err := escenario.repositorio.ConfirmarCambio(
				context.Background(), confirmacion,
			); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
				t.Fatalf("estructura historica corrupta admitida: %v", err)
			}
			comprobarLongitudesInternas(t, escenario.repositorio, 3, 3, 3)
		})
	}
}
