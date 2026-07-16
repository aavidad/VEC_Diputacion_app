package memory

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func huellaTokenReservaCotejoPrueba(t *testing.T, token ports.TokenReservaEmisionCodigoCotejo) string {
	t.Helper()
	huella, err := token.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella del token de reserva de cotejo: %v", err)
	}
	return huella
}

func TestCotejoMemoriaReservaEsIdempotenteExpiraYConfirmaSinParcialidad(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	ahora := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	documento := domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-001", Version: 1}
	solicitud := cotejoMemoriaPruebaSolicitudReserva(
		"idempotencia-cotejo-memoria-reserva-001", documento, cotejoMemoriaPruebaSolicitudA, ahora,
	)

	primera, err := store.ReservarEmisionCodigoCotejo(ctx, solicitud)
	if err != nil || !primera.Token.Valido() || primera.Repetida {
		t.Fatalf("primera reserva = %+v, %v", primera, err)
	}
	internaPrimera := store.reservasCotejo[claveAmbitoCotejo(solicitud.PrincipalID, solicitud.ClaveIdempotencia)]
	if internaPrimera.HuellaTokenSHA256 != huellaTokenReservaCotejoPrueba(t, primera.Token) {
		t.Fatalf("la reserva no conservo solo la huella esperada: %+v", internaPrimera)
	}
	if _, conservaToken := reflect.TypeOf(internaPrimera).FieldByName("Token"); conservaToken {
		t.Fatal("el estado interno conserva el token de cotejo en claro")
	}
	enCurso := solicitud
	enCurso.SolicitadaEn = ahora.Add(time.Minute)
	enCurso.ExpiraEn = enCurso.SolicitadaEn.Add(5 * time.Minute)
	if _, err := store.ReservarEmisionCodigoCotejo(ctx, enCurso); !errors.Is(err, ports.ErrEmisionCodigoCotejoEnCurso) {
		t.Fatalf("repeticion antes de caducar: error = %v", err)
	}
	reutilizada := enCurso
	reutilizada.HuellaSolicitudHMAC = cotejoMemoriaPruebaSolicitudB
	if _, err := store.ReservarEmisionCodigoCotejo(ctx, reutilizada); !errors.Is(err, ports.ErrClaveIdempotenciaCotejoReutilizada) {
		t.Fatalf("reutilizacion con otro significado: error = %v", err)
	}

	reintento := solicitud
	reintento.SolicitadaEn = solicitud.ExpiraEn
	reintento.ExpiraEn = reintento.SolicitadaEn.Add(5 * time.Minute)
	segunda, err := store.ReservarEmisionCodigoCotejo(ctx, reintento)
	if err != nil || !segunda.Token.Valido() ||
		huellaTokenReservaCotejoPrueba(t, segunda.Token) == huellaTokenReservaCotejoPrueba(t, primera.Token) {
		t.Fatalf("reserva tras expiracion = %+v, %v", segunda, err)
	}
	if _, existe := store.reservasCotejoPorHuellaToken[huellaTokenReservaCotejoPrueba(t, primera.Token)]; existe {
		t.Fatal("el token expirado sigue siendo confirmable")
	}

	codigo := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-001", documento, cotejoMemoriaPruebaIndiceA,
		cotejoMemoriaPruebaProteccionA, reintento.SolicitadaEn,
	)
	cotejoMemoriaPruebaSembrarDependenciasReserva(store, codigo)
	confirmadaEn := codigo.ReservadoEn.Add(time.Minute)
	traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(codigo, domain.AccionCodigoCotejoReservado, "", confirmadaEn)
	traza.Metadata["valor_csv"] = cotejoMemoriaPruebaSecreto
	if err := store.ConfirmarReservaCodigoCotejo(
		ctx, segunda.Token, reintento.HuellaSolicitudHMAC, confirmadaEn, codigo, traza, evento,
	); !errors.Is(err, ports.ErrReservaCodigoCotejoNoValida) {
		t.Fatalf("confirmacion con auditoria insegura: error = %v", err)
	}
	if len(store.codigosCotejo) != 0 || len(store.cotejoPorDocumento) != 0 || len(store.cotejoPorIndice) != 0 ||
		len(store.audit) != 0 || len(store.events) != 0 {
		t.Fatalf("la confirmacion fallida dejo estado parcial: codigos=%d documentos=%d indices=%d auditoria=%d eventos=%d",
			len(store.codigosCotejo), len(store.cotejoPorDocumento), len(store.cotejoPorIndice), len(store.audit), len(store.events))
	}
	if _, existe := store.reservasCotejoPorHuellaToken[huellaTokenReservaCotejoPrueba(t, segunda.Token)]; !existe {
		t.Fatal("el fallo no atomico consumio el token valido")
	}

	traza, evento = cotejoMemoriaPruebaEvidenciaCodigo(codigo, domain.AccionCodigoCotejoReservado, "", confirmadaEn)
	if err := codigo.Validar(); err != nil || !evidenciaCodigoCotejoValida(
		codigo, traza, evento, domain.AccionCodigoCotejoReservado, "", confirmadaEn,
	) {
		t.Fatalf("fixture de confirmacion invalido: codigo=%v traza=%+v evento=%v", err, traza, evento)
	}
	if err := store.ConfirmarReservaCodigoCotejo(
		ctx, segunda.Token, reintento.HuellaSolicitudHMAC, confirmadaEn, codigo, traza, evento,
	); err != nil {
		t.Fatalf("confirmar reserva valida: %v", err)
	}
	if err := store.ConfirmarReservaCodigoCotejo(
		ctx, segunda.Token, reintento.HuellaSolicitudHMAC, confirmadaEn, codigo, traza, evento,
	); !errors.Is(err, ports.ErrReservaCodigoCotejoNoValida) {
		t.Fatalf("replay de confirmacion: error=%v", err)
	}
	if err := store.AbandonarReservaCodigoCotejo(ctx, segunda.Token); !errors.Is(err, ports.ErrReservaCodigoCotejoNoValida) {
		t.Fatalf("replay de abandono tras confirmacion: error=%v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
	if _, existe := store.reservasCotejoPorHuellaToken[huellaTokenReservaCotejoPrueba(t, segunda.Token)]; existe {
		t.Fatal("el token confirmado no fue consumido")
	}

	repeticion := reintento
	repeticion.SolicitadaEn = ahora.Add(20 * time.Minute)
	repeticion.ExpiraEn = repeticion.SolicitadaEn.Add(5 * time.Minute)
	repetida, err := store.ReservarEmisionCodigoCotejo(ctx, repeticion)
	if err != nil || !repetida.Repetida || repetida.Token.Valido() || !reflect.DeepEqual(repetida.Codigo, codigo) {
		t.Fatalf("repeticion confirmada = %+v, %v", repetida, err)
	}
	repetida.Codigo.MotivoReserva = "mutacion externa"
	guardado, err := store.ObtenerCodigoCotejo(ctx, codigo.ID)
	if err != nil || guardado.MotivoReserva == "mutacion externa" {
		t.Fatalf("la repeticion comparte memoria con el almacen: %+v, %v", guardado, err)
	}
}
