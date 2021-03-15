package pdf

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/httpimg"
	"github.com/metal3d/go-slugify"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/utils"
)

// Report asdf
func (d *Delivery) CreateDeliveryPDF(outputDir string) (string, error) {
	errPath := "pdf.CreateReport()"
	now := time.Time.Format(time.Now(), "2006-01-02")
	pdfName := fmt.Sprintf("%s_%s.pdf", now, slugify.Marshal(d.OrgName))
	filename := filepath.Join(outputDir, pdfName)
	pdf := gofpdf.New("P", "mm", "Letter", "")

	pdf.AliasNbPages("")
	pdf.SetAutoPageBreak(true, 40)

	pdf.SetFooterFunc(func() {
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		pdf.SetXY(120, 250)
		bodyfont(pdf)
		pdf.SetLineWidth(0.4)
		pdf.CellFormat(80, 12, tr("Recibe "+d.Receiver), "T", 0, "C", false, 0, "")
		pdf.SetXY(20, 260)
		pdf.SetFont("Helvetica", "", 7)
		pdf.CellFormat(0, 12, tr("Av. Vitacura 5362 - Of.A, Vitacura, Santiago, Chile  T: +56 (2) 9805352 T: +56-(2)-3210-0951 - soporte@cloner.cl - www.cloner.cl"), "", 0, "C", false, 0, "")
	})

	//Genera el contenido de cada pagina
	makeContent("Cloner", d, pdf)
	makeContent("Cliente", d, pdf)
	if err := pdf.OutputFileAndClose(filename); err != nil {
		log.Error("%s", errors.New(errPath, err))
	}
	log.Task("Wrote delivery PDF to: %s", filename)
	return filename, nil
}

func makeContent(copy string, d *Delivery, pdf *gofpdf.Fpdf) {
	pdf.AddPage()
	makeHeader(copy, pdf)        // Header
	makeParagraph1(d, pdf)       //Primer Parrafo
	makeRecoveryTable(d, pdf)    //Tabla de detalle de recuperaciones
	makeDiskstable(d, pdf)       //Tabla de detalle de discos
	makeResponsibilities(d, pdf) //Texto final de responsabilidades
}
func makeHeader(copy string, pdf *gofpdf.Fpdf) {
	now := time.Time.Format(time.Now(), "2006-01-02")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetXY(10, 10)
	url := "https://soporte.cloner.cl/images/LogoClonerM.png"
	httpimg.Register(pdf, url, "")
	pdf.Image(url, 20, 10, 26.4979, 35, false, "", 0, "")
	pdf.SetXY(20, 45)
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetDrawColor(220, 220, 220)
	pdf.SetLineWidth(0.4)
	pdf.CellFormat(176, 9, now, "T", 0, "R", false, 0, "") //Fecha y separador
	pdf.SetXY(55, 15)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cellf(210, 10, tr("RECUPERACIÓN DE INFORMACIÓN")) //Recuperación de información (Header)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetXY(165, 15)
	copy = fmt.Sprintf("(Copia %s)", copy)
	pdf.Cellf(100, 10, tr(copy)) // copia cloner / cliente
	pdf.SetXY(90, 30)
	pdf.SetFont("Helvetica", "I", 14)
	pdf.SetTextColor(80, 80, 80)
	msg := youAreSafe()
	pdf.CellFormat(100, 10, tr(msg), "", 0, "R", false, 0, "") // Mensaje Cloner
}
func makeRecoveryTable(d *Delivery, pdf *gofpdf.Fpdf) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	//table header
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.SetLineWidth(0)
	tablefont(pdf)
	tableheader(pdf)
	pdf.CellFormat(58, cellheight, "Usuario", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, cellheight, "Dispositivo", "1", 0, "C", true, 0, "")
	pdf.CellFormat(58, cellheight, "Recuperado", "1", 1, "C", true, 0, "")
	// Cuerpo de tabla
	even := false
	for _, rec := range d.Recoveries {
		pdf.SetDrawColor(200, 200, 200)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetX(20)
		if even {
			pdf.SetFillColor(255, 255, 255)
			pdf.CellFormat(58, cellheight, tr(rec.User), "", 0, "C", true, 0, "")
			pdf.CellFormat(60, cellheight, tr(rec.Machine), "L", 0, "C", true, 0, "")
			pdf.CellFormat(58, cellheight, utils.B2H(rec.Size), "L", 1, "C", true, 0, "")
		} else {
			pdf.SetFillColor(220, 220, 220)
			pdf.CellFormat(58, cellheight, tr(rec.User), "", 0, "C", true, 0, "")
			pdf.CellFormat(60, cellheight, tr(rec.Machine), "L", 0, "C", true, 0, "")
			pdf.CellFormat(58, cellheight, utils.B2H(rec.Size), "L", 1, "C", true, 0, "")
		}
		even = !even
	}
	pdf.SetX(78) //total final de la tabla
	pdf.SetDrawColor(32, 162, 126)
	if even {
		pdf.SetFillColor(255, 255, 255)
		pdf.CellFormat(60, cellheight, "Total", "T", 0, "C", true, 0, "")
		pdf.CellFormat(58, cellheight, utils.B2H(d.TotalSize), "T", 1, "C", true, 0, "")
	} else {
		pdf.SetFillColor(220, 220, 220)
		pdf.CellFormat(60, cellheight, "Total", "T", 0, "C", true, 0, "")
		pdf.CellFormat(58, cellheight, utils.B2H(d.TotalSize), "T", 1, "C", true, 0, "")
	}
	pdf.Ln(5)
	setParagraph(pdf)
}

func makeDiskstable(d *Delivery, pdf *gofpdf.Fpdf) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	bodyfont(pdf)
	setParagraph(pdf)
	pdf.MultiCell(176, lineheight, tr("La información recuperada se cargó en el siguiente hardware para el envío:"), "", "J", false)
	//table header
	pdf.Ln(5) // espaciador
	pdf.SetX(20)
	tablefont(pdf)
	tableheader(pdf)
	pdf.SetLineWidth(0)
	pdf.CellFormat(40, cellheight, "Disco", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, cellheight, "Marca", "1", 0, "C", true, 0, "")
	pdf.CellFormat(56, cellheight, "Numero de serie", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, cellheight, "Capacidad", "1", 1, "C", true, 0, "")
	// Cuerpo de tabla
	even := false
	for _, disk := range d.Disks {
		pdf.SetDrawColor(200, 200, 200)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetX(20)
		if even {
			pdf.SetFillColor(255, 255, 255)
			pdf.CellFormat(40, cellheight, disk.Name, "", 0, "C", true, 0, "")
			pdf.CellFormat(40, cellheight, disk.Brand, "L", 0, "C", true, 0, "")
			pdf.CellFormat(56, cellheight, disk.Serial, "L", 0, "C", true, 0, "")
			pdf.CellFormat(40, cellheight, disk.Size, "L", 1, "C", true, 0, "")
		} else {
			pdf.SetFillColor(220, 220, 220)
			pdf.CellFormat(40, cellheight, disk.Name, "", 0, "C", true, 0, "")
			pdf.CellFormat(40, cellheight, disk.Brand, "L", 0, "C", true, 0, "")
			pdf.CellFormat(56, cellheight, disk.Serial, "L", 0, "C", true, 0, "")
			pdf.CellFormat(40, cellheight, disk.Size, "L", 1, "C", true, 0, "")
		}
		even = !even
	}
	pdf.Ln(5)
}

func makeResponsibilities(d *Delivery, pdf *gofpdf.Fpdf) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	setParagraph(pdf)
	if len(d.Disks) == 1 {
		pdf.MultiCell(176, lineheight, tr("Al hacer entrega de este disco duro, la organización "+d.OrgName+" se hace responsable de la información contenida en este y también de la integridad del artefacto, el cual debe ser devuelto en el mismo estado al cual fue entregado y con su cable conector.\n\nEl hardware se entrega en modalidad de préstamo por un plazo máximo de 10 días hábiles. La devolución es de exclusiva responsabilidad de la empresa "+
			d.OrgName+" . De forma opcional la empresa puede comprar el disco duro por un valor de "+strconv.Itoa(d.Disks[0].Value)+
			" UF, no teniendo que devolverlo"+".\n\n"), "", "J", false)
	} else {
		var disktotalVal int
		for _, disk := range d.Disks {
			disktotalVal += disk.Value
		}
		pdf.MultiCell(176, lineheight, tr("Al hacer entrega de estos discos duros, la organización "+d.OrgName+" se hace responsable de la información contenida en estos y también de la integridad de los artefactos, los cuales deben ser devuelto en el mismo estado al cual fue entregados y con su cables conectores.\n\nEl hardware se entrega en modalidad de préstamo por un plazo máximo de 10 días hábiles. La devolución es de exclusiva responsabilidad de la empresa "+
			d.OrgName+". Opcionalmente la empresa puede comprar los discos duros por un valor total de "+strconv.Itoa(disktotalVal)+" UF, no teniendo que devolverlos"+"."), "", "J", false)
	}
}

func youAreSafe() string { //mensajes promocionales :D
	x := []string{
		"Tu información está segura con nosotros",
		"Tus datos, enviados diréctamente a tu oficina",
		"Los virus no son un problema si estás con nosotros",
		"Las perdidas no son un problema si estás con nosotros",
		"La perdida de un equipo no es un problema si estás con nosotros",
		"No borramos tu información, siempre podrás recuperarla",
		"Siempre respaldaremos tu información",
		"Nuestro equipo está para responder tus dudas",
		"Respalda en nuestra nube, estamos para apoyarte",
		"Recupera tu información, mas rápido que nunca",
		"Con nosotros, no perderás tu datos",
	}
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(len(x))
	return x[r]
}

func makeParagraph1(d *Delivery, pdf *gofpdf.Fpdf) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	// Contenido General
	pdf.SetXY(20, 55)
	bodyfont(pdf)
	pdf.MultiCell(176, lineheight, tr("Mediante el presente documento la empresa Cloner SpA, a través de su empleado "+d.ExtDelivery+
		", hace entrega de la recuperación solicitada por el usuario "+d.Requester+" de la organización "+d.OrgName+"."+
		"\n\nA continuación se detalla la recuperación solicitada la cual se envía a la dirección "+d.Address+", para ser recibida por "+d.Receiver), "", "J", false)
}

func bodyfont(pdf *gofpdf.Fpdf) {
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(0, 0, 0)
}
func tablefont(pdf *gofpdf.Fpdf) {
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(0, 0, 0)
}
func tableheader(pdf *gofpdf.Fpdf) {
	pdf.SetFillColor(32, 162, 126)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetDrawColor(255, 255, 255)
}
func setParagraph(pdf *gofpdf.Fpdf) {
	pdf.SetX(20)
	bodyfont(pdf)
}
