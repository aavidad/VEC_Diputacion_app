//go:build ignore && linux && amd64

package main

import (
	"errors"
	"os"
	"strconv"
	"syscall"
	"time"
)

const (
	maximoTicketO3bM38 = 2048
	maximoTramaO3bM38  = 2060
)

const operacionEscribirTicketO3bM38 operacionGuardiaO3aM38 = operacionBarreraO3bM38 + 1

var (
	errTicketO3bM38       = errors.New("ticket O3b no acreditado")
	errEscrituraO3bM38    = errors.New("escritura de ticket O3b fallida")
	errCierreTicketO3bM38 = errors.New("cierre de ticket O3b fallido")
)

type ticketPreparadoO3bM38 struct {
	trama           [maximoTramaO3bM38]byte
	longitud        int
	fd              int
	trasCierre      snapshotFDO3aM38
	emisionIniciada bool
	cierreIntentado bool
	primerPermiso   permisoGuardiaO3aM38
	permisoPrimero  bool
}

func ticketRetenidoO3bM38(c *custodiaO3aM38) (string, bool) {
	if c == nil || c.control == nil || c.control.recepcion == nil {
		return "", false
	}
	ticket := c.control.recepcion.sobre.ticket
	if len(ticket) < 1 || len(ticket) > maximoTicketO3bM38 {
		return "", false
	}
	for i := 0; i < len(ticket); i++ {
		if ticket[i] < 0x20 || ticket[i] > 0x7e {
			return "", false
		}
	}
	return ticket, true
}

func copiarSnapshotSinFDO3bM38(origen snapshotFDO3aM38, fd int) (snapshotFDO3aM38, bool) {
	if origen.mapa == nil {
		return snapshotFDO3aM38{}, false
	}
	if huella, existe := origen.mapa[fd]; !existe || !huella.abierto {
		return snapshotFDO3aM38{}, false
	}
	copia := snapshotFDO3aM38{limite: origen.limite, mapa: make(map[int]huellaFDO3aM38, len(origen.mapa)-1)}
	for actual, huella := range origen.mapa {
		if actual != fd {
			copia.mapa[actual] = huella
		}
	}
	return copia, true
}

// prepararTicketO3bM38 se ejecuta dentro de P2, antes de su última relectura.
// Toda reserva y copia queda así anterior al verde B1.
func prepararTicketO3bM38(a *autoridadCapturaO3bM38) error {
	if a == nil || a.estado != capturaB0RecibidoM38 || a.custodia == nil || a.ticket != nil ||
		a.custodia.ticketEscritor == nil {
		return errTicketO3bM38
	}
	fd := int(a.custodia.ticketEscritor.Fd())
	if fd < 0 || fd != a.fdsBarrera[2] {
		return errTicketO3bM38
	}
	trasCierre, valido := copiarSnapshotSinFDO3bM38(a.custodia.lease.fisico, fd)
	if !valido {
		return errTicketO3bM38
	}
	preparado := &ticketPreparadoO3bM38{fd: fd, trasCierre: trasCierre}
	// Desde este punto la retirada posee toda la información necesaria para
	// separar y cerrar el escritor incluso si la trama resulta inválida.
	a.ticket = preparado
	ticket, valido := ticketRetenidoO3bM38(a.custodia)
	if !valido {
		return errTicketO3bM38
	}
	trama := strconv.AppendInt(preparado.trama[:0], int64(os.Getpid()), 10)
	trama = append(trama, '|')
	trama = append(trama, ticket...)
	trama = append(trama, '\n')
	if len(trama) < 4 || len(trama) > len(preparado.trama) {
		return errTicketO3bM38
	}
	preparado.longitud = len(trama)
	return nil
}

func aplicarResultadoEscrituraO3bM38(restantes, n, interrupciones int, err error) (avance, nuevasInterrupciones int, fallo error) {
	if n < 0 || n > restantes {
		return 0, interrupciones, errEscrituraO3bM38
	}
	if errors.Is(err, syscall.EINTR) {
		interrupciones++
		if interrupciones > 8 {
			return n, interrupciones, syscall.EINTR
		}
		return n, interrupciones, nil
	}
	if err != nil {
		return 0, interrupciones, err
	}
	if n == 0 {
		return 0, interrupciones, errEscrituraO3bM38
	}
	return n, interrupciones, nil
}

func escribirTicketO3bM38(a *autoridadCapturaO3bM38) error {
	preparado := a.ticket
	if preparado == nil || preparado.emisionIniciada || preparado.longitud <= 0 || preparado.longitud > len(preparado.trama) {
		return errTicketO3bM38
	}
	preparado.emisionIniciada = true
	offset, interrupciones := 0, 0
	primerIntento := true
	for offset < preparado.longitud {
		// La barrera ya acreditó el plazo. El primer syscall posterior a B1 ha
		// de ser Write; el reloj solo se relee antes de sus reintentos.
		if !primerIntento && !time.Now().Before(a.custodia.finBootstrap) {
			return errEscrituraO3bM38
		}
		esPrimero := primerIntento
		primerIntento = false
		n := 0
		var err error
		if esPrimero {
			// El permiso se creó como último paso falible de la barrera. Así el
			// primer syscall ejecutado después de B1 es literalmente Write.
			n, err = syscall.Write(preparado.fd, preparado.trama[offset:preparado.longitud])
			if !preparado.permisoPrimero || !a.custodia.lease.consolidarCritico(preparado.primerPermiso) {
				fatalBarreraO3bM38(a)
				select {}
			}
			preparado.permisoPrimero = false
		} else {
			err = operarConLeaseBarreraO3bM38(a.custodia, func() error {
				var errEscritura error
				n, errEscritura = syscall.Write(preparado.fd, preparado.trama[offset:preparado.longitud])
				return errEscritura
			})
		}
		if errors.Is(err, errLeaseBarreraO3bM38) {
			fatalBarreraO3bM38(a)
			select {}
		}
		avance, nuevasInterrupciones, fallo := aplicarResultadoEscrituraO3bM38(preparado.longitud-offset, n, interrupciones, err)
		interrupciones = nuevasInterrupciones
		if fallo != nil {
			return fallo
		}
		offset += avance
	}
	return nil
}

func cerrarTicketO3bM38(a *autoridadCapturaO3bM38) error {
	preparado := a.ticket
	if preparado == nil || preparado.cierreIntentado || a.custodia.ticketEscritor == nil {
		return errCierreTicketO3bM38
	}
	archivo := a.custodia.ticketEscritor
	a.custodia.ticketEscritor = nil
	preparado.cierreIntentado = true
	permiso, valido := a.custodia.lease.comenzar(operacionCerrarTicketO3aM38, 1, [2]int{preparado.fd, -1})
	if !valido {
		fatalBarreraO3bM38(a)
		select {}
	}
	errCierre := archivo.Close()
	if !a.custodia.lease.consolidarFisico(permiso, preparado.trasCierre, true) {
		fatalBarreraO3bM38(a)
		select {}
	}
	if errCierre != nil {
		return errCierre
	}
	return nil
}

func retirarTicketO3bM38(a *autoridadCapturaO3bM38, primera error) error {
	if a != nil && a.custodia != nil && a.custodia.ticketEscritor != nil && a.ticket == nil {
		fatalBarreraO3bM38(a)
		select {}
	}
	if a != nil && a.ticket != nil && !a.ticket.cierreIntentado && a.custodia != nil && a.custodia.ticketEscritor != nil {
		_ = cerrarTicketO3bM38(a)
	}
	if a != nil && transicionCapturaO3bM38(a.estado, capturaB7RetirandoM38) {
		a.estado = capturaB7RetirandoM38
	}
	return primera
}

// emitirYCerrarTicketO3bM38 consume el único escritor preparado. No observa
// STOP, no transfiere autoridades y no vuelve a escribir durante la retirada.
func emitirYCerrarTicketO3bM38(a *autoridadCapturaO3bM38) error {
	if a == nil || a.estado != capturaB1BarreraVerdeM38 || a.custodia == nil {
		return retirarTicketO3bM38(a, errTicketO3bM38)
	}
	if a.ticket == nil {
		fatalBarreraO3bM38(a)
		select {}
	}
	if err := escribirTicketO3bM38(a); err != nil {
		return retirarTicketO3bM38(a, err)
	}
	if err := cerrarTicketO3bM38(a); err != nil {
		return retirarTicketO3bM38(a, err)
	}
	if !transicionCapturaO3bM38(a.estado, capturaB2TicketCerradoM38) {
		return retirarTicketO3bM38(a, errTicketO3bM38)
	}
	a.estado = capturaB2TicketCerradoM38
	return nil
}
