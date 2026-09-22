package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// cifradorDatosContactoPrueba invierte los bytes: suficiente para comprobar que el
// claro nunca llega al repositorio y que la lectura descifra lo guardado.
type cifradorDatosContactoPrueba struct{ cifrados, descifrados int }

func invertirDatosContacto(b []byte) []byte {
	r := make([]byte, len(b))
	for i := range b {
		r[len(b)-1-i] = b[i]
	}
	return r
}

func (c *cifradorDatosContactoPrueba) CifrarDatosContactoParticipacion(_ context.Context, ref string, version uint64, claro []byte) (puertosbolsa.SobreDatosContacto, error) {
	c.cifrados++
	return puertosbolsa.SobreDatosContacto{Version: version, ClaveRef: "clave:prueba", Nonce: []byte(ref), Cifrado: invertirDatosContacto(claro)}, nil
}

func (c *cifradorDatosContactoPrueba) ConDatosContactoParticipacionDescifrados(_ context.Context, ref string, sobre puertosbolsa.SobreDatosContacto, usar func([]byte) error) error {
	c.descifrados++
	if !bytes.Equal(sobre.Nonce, []byte(ref)) {
		return errors.New("sobre de otra participación")
	}
	return usar(invertirDatosContacto(sobre.Cifrado))
}

type repositorioDatosContactoPrueba struct {
	registros []puertosbolsa.RegistroDatosContactoParticipacion
	llamadas  int
	ultimo    puertosbolsa.ComandoRegistrarDatosContactoParticipacion
}

func (r *repositorioDatosContactoPrueba) DatosContactoVigentes(_ context.Context, ref string) (puertosbolsa.RegistroDatosContactoParticipacion, error) {
	for i := len(r.registros) - 1; i >= 0; i-- {
		if r.registros[i].ParticipacionRef == ref {
			return r.registros[i], nil
		}
	}
	return puertosbolsa.RegistroDatosContactoParticipacion{}, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados
}

func (r *repositorioDatosContactoPrueba) BuscarRegistroDatosContacto(_ context.Context, ref, clave string) (puertosbolsa.RegistroDatosContactoParticipacion, error) {
	for _, registro := range r.registros {
		if registro.ParticipacionRef == ref && registro.ReciboRef == reciboDatosContactoPrueba(ref, clave) {
			return registro, nil
		}
	}
	return puertosbolsa.RegistroDatosContactoParticipacion{}, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados
}

func (r *repositorioDatosContactoPrueba) RegistrarDatosContacto(_ context.Context, comando puertosbolsa.ComandoRegistrarDatosContactoParticipacion) (puertosbolsa.RegistroDatosContactoParticipacion, error) {
	r.llamadas++
	r.ultimo = comando
	registro := puertosbolsa.RegistroDatosContactoParticipacion{ReciboRef: comando.ReciboRef, ParticipacionRef: comando.ParticipacionRef, Version: comando.Sobre.Version, Motivo: comando.Motivo, RegistradaEn: comando.RegistradaEn, Sobre: comando.Sobre}
	r.registros = append(r.registros, registro)
	return registro, nil
}

// reciboDatosContactoPrueba reproduce el recibo determinista del servicio.
func reciboDatosContactoPrueba(ref, clave string) string {
	h := sha256.Sum256([]byte(ref + "\x1f" + clave))
	return "recibo:datos-contacto:" + hex.EncodeToString(h[:])
}

type pertenenciaDatosContactoPrueba struct{ pertenece bool }

func (p pertenenciaDatosContactoPrueba) ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error) {
	return p.pertenece, nil
}

func solicitudDatosContactoPrueba(t *testing.T, ahora time.Time, clave string, datos dominiobolsa.DatosContactoParticipacion) puertosbolsa.SolicitudRegistrarDatosContactoParticipacion {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return puertosbolsa.SolicitudRegistrarDatosContactoParticipacion{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:b4", ParticipacionRef: datos.ParticipacionRef, Datos: datos, Motivo: "Alta de contacto comunicada por el candidato", ClaveIdempotencia: clave, Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()}
}

func datosDatosContactoPrueba() dominiobolsa.DatosContactoParticipacion {
	return dominiobolsa.DatosContactoParticipacion{ParticipacionRef: "participacion:b4", Correo: "Candidata@Dipgra.ES", Telefono1: "600 123 456", Telefono2: "958 24 70 00"}
}

func servicioDatosContactoPrueba(t *testing.T, ahora time.Time, repo *repositorioDatosContactoPrueba, cifrador *cifradorDatosContactoPrueba, contexto puertosbolsa.ResolutorContextoSituacionParticipacion, pertenece bool) (*ServicioDatosContactoParticipacion, *autorizadorBorradorPrueba) {
	t.Helper()
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioDatosContactoParticipacion(contexto, autorizador, pertenenciaDatosContactoPrueba{pertenece}, cifrador, repo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	return servicio, autorizador
}

func TestServicioContactoRegistraCifradoYConsultaDescifrado(t *testing.T) {
	ahora := time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)
	repo := &repositorioDatosContactoPrueba{}
	cifrador := &cifradorDatosContactoPrueba{}
	servicio, autorizador := servicioDatosContactoPrueba(t, ahora, repo, cifrador, contextoSituacionPrueba{}, true)
	registro, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0001", datosDatosContactoPrueba()))
	if err != nil || registro.Reutilizada || registro.Version != 1 || repo.llamadas != 1 || autorizador.llamadas != 1 || cifrador.cifrados != 1 {
		t.Fatalf("registro=%+v err=%v repo=%d auth=%d cifrados=%d", registro, err, repo.llamadas, autorizador.llamadas, cifrador.cifrados)
	}
	if bytes.Contains(repo.ultimo.Sobre.Cifrado, []byte("dipgra.es")) || bytes.Contains(repo.ultimo.Sobre.Cifrado, []byte("600123456")) {
		t.Fatal("el claro no puede llegar al repositorio")
	}
	if repo.ultimo.Actor != "per_0123456789abcdefghijkl" || repo.ultimo.RegistradaEn != ahora || repo.ultimo.Material.ValidarEstructura() != nil {
		t.Fatalf("comando: %+v", repo.ultimo)
	}
	leidos, err := servicio.Consultar(context.Background(), puertosbolsa.SolicitudConsultarDatosContactoParticipacion{ContextoActor: dominiovec.ContextoActor{PersonaRef: "per_0123456789abcdefghijkl"}, BolsaRef: "bolsa:b4", ParticipacionRef: "participacion:b4"})
	if err != nil || leidos.Version != 1 || leidos.Datos.Correo != "Candidata@dipgra.es" || leidos.Datos.Telefono1 != "600123456" || leidos.Datos.Telefono2 != "958247000" || leidos.Enmascarados.Telefono1 != "***3456" {
		t.Fatalf("lectura=%+v err=%v", leidos, err)
	}
}

func TestServicioContactoRepiteSinVersionNuevaYRechazaOtrosDatos(t *testing.T) {
	ahora := time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)
	repo := &repositorioDatosContactoPrueba{}
	cifrador := &cifradorDatosContactoPrueba{}
	servicio, _ := servicioDatosContactoPrueba(t, ahora, repo, cifrador, contextoSituacionPrueba{}, true)
	primero, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0001", datosDatosContactoPrueba()))
	if err != nil {
		t.Fatal(err)
	}
	repetido, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0001", datosDatosContactoPrueba()))
	if err != nil || !repetido.Reutilizada || repetido.ReciboRef != primero.ReciboRef || repo.llamadas != 1 {
		t.Fatalf("repetición: %+v err=%v llamadas=%d", repetido, err, repo.llamadas)
	}
	otros := datosDatosContactoPrueba()
	otros.Telefono1 = "611111111"
	if _, err = servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0001", otros)); !errors.Is(err, dominiobolsa.ErrDatosContactoParticipacionInvalidos) {
		t.Fatalf("misma clave con otros datos debe rechazarse: %v", err)
	}
	segundo, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0002", otros))
	if err != nil || segundo.Version != 2 || repo.llamadas != 2 {
		t.Fatalf("segunda versión: %+v err=%v llamadas=%d", segundo, err, repo.llamadas)
	}
}

func TestServicioContactoDeniegaSinAmbitoOParticipacionAjenaAntesDeCifrar(t *testing.T) {
	ahora := time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)
	repo := &repositorioDatosContactoPrueba{}
	cifrador := &cifradorDatosContactoPrueba{}
	servicio, autorizador := servicioDatosContactoPrueba(t, ahora, repo, cifrador, contextoSituacionPrueba{err: dominiovec.ErrAutorizacionDenegada}, true)
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0001", datosDatosContactoPrueba())); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || autorizador.llamadas != 0 || cifrador.cifrados != 0 || repo.llamadas != 0 {
		t.Fatalf("sin ámbito: err=%v auth=%d cifrados=%d repo=%d", err, autorizador.llamadas, cifrador.cifrados, repo.llamadas)
	}
	servicio, autorizador = servicioDatosContactoPrueba(t, ahora, repo, cifrador, contextoSituacionPrueba{}, false)
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0001", datosDatosContactoPrueba())); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || autorizador.llamadas != 0 || cifrador.cifrados != 0 {
		t.Fatalf("participación ajena: err=%v auth=%d cifrados=%d", err, autorizador.llamadas, cifrador.cifrados)
	}
	if _, err := servicio.Consultar(context.Background(), puertosbolsa.SolicitudConsultarDatosContactoParticipacion{ContextoActor: dominiovec.ContextoActor{PersonaRef: "per_0123456789abcdefghijkl"}, BolsaRef: "bolsa:b4", ParticipacionRef: "participacion:b4"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || cifrador.descifrados != 0 {
		t.Fatalf("lectura ajena: err=%v descifrados=%d", err, cifrador.descifrados)
	}
	invalidos := datosDatosContactoPrueba()
	invalidos.Correo, invalidos.Telefono1, invalidos.Telefono2 = "", "", ""
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "b4-alta-0009", invalidos)); !errors.Is(err, dominiobolsa.ErrDatosContactoParticipacionInvalidos) {
		t.Fatalf("datos vacíos: %v", err)
	}
}
