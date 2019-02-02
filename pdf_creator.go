package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"

	"github.com/jung-kurt/gofpdf"
)

//Data to use in charts and tables. DataPoints is an interface since we don't want to hardcode the names of the categories or series values
type Data []struct {
	DataSource string        `json dataSource`
	DataPoints []interface{} `json: dataPoints`
}

//Chart settings
type ChartSettings struct {
	WatermarkFormat               ShapeStyle `json: watermarkFormat`
	SeriesFormat                  ShapeStyle `json: seriesFormat`
	AxisFormat                    ShapeStyle `json: axisFormat`
	ChartTextFont                 Font       `json: chartTextFont`
	ChartTitle                    ChartTitle `json: chartTitle`
	DistanceFromTopOfChartArea    float64    `json: distanceFromTopOfChartArea`
	DistanceFromBottomOfChartArea float64    `json: distanceFromBottomOfChartArea`
	DistanceFromSidesOfChartArea  float64    `json: distanceFromSidesOfChartArea`
	NumberOfYAxisTicks            float64    `json: numberOfYAxisTicks`
	GapBetweenBars                float64    `json: gapBetweenBars`
	TickMarkLength                float64    `json: tickMarkLength`
}

//Mapping the fields from the recipe - that describes how the pdf is built and its contents
type PdfFields struct {
	PdfSettings struct {
		PageOrientation         string  `json: pageOrientation`
		PageUnits               string  `json: pageUnits`
		PdfName                 string  `json: pdfName`
		PdfLocation             string  `json: pdfLocation`
		PageHeight              float64 `json: pageHeight`
		PageWidth               float64 `json: pageWidth`
		PageLeftAndRightMargins float64 `json: pageLeftAndRightMargins`
		PageTopMargin           float64 `json: pageTopMargin`
		Watermark               Colour  `json: watermark`
	} `json: pdfSettings`
	PdfContents []PdfContentItem `json: pdfContents`
}

//Refers to the pdf items that are written to the pages. Examples would be tables, vertical bar charts etc
type PdfContentItem struct {
	ItemType           string        `json: itemType`
	Text               string        `json: text`
	DataSource         string        `json dataSource`
	DataSeries         string        `json: dataSeries`
	DataSeriesCategory string        `json: dataSeriesCategory`
	XPosition          float64       `json: xPosition`
	YPosition          float64       `json: yPosition`
	Width              float64       `json: width`
	Height             float64       `json: height`
	Font               Font          `json: font`
	ChartSettings      ChartSettings `json: chartSettings`
}

type Font struct {
	Colour      Colour      `json: colour`
	Style       string      `json: style`
	Size        float64     `json: size`
	Family      string      `json: family`
	Alignment   string      `json: alignment`
	LineSpacing float64     `json: lineSpacing`
	CellBorders CellBorders `json: cellBorders`
	CellFill    CellFill    `json: cellFill`
	HeaderFont  HeaderFont  `json: headerFont`
}

type HeaderFont struct {
	Colour      Colour      `json: colour`
	Style       string      `json: style`
	Size        float64     `json: size`
	Family      string      `json: family`
	Alignment   string      `json: alignment`
	LineSpacing float64     `json: lineSpacing`
	CellBorders CellBorders `json: cellBorders`
	CellFill    CellFill    `json: cellFill`
}

type CellBorders struct {
	Style  string `json: style`
	Colour Colour `json: colour`
}

type CellFill struct {
	Filled bool   `json: filled`
	Colour Colour `json: colour`
}

type ShapeStyle struct {
	Style        string  `json: style`
	FillColour   Colour  `json: fillColour`
	BorderColour Colour  `json: borderColour`
	LineWidth    float64 `json: lineWidth`
	LineColour   Colour  `json: lineColour`
}

type ChartTitle struct {
	Text                       string  `json: text`
	DistanceFromTopOfChartArea float64 `json: distanceFromTopOfChartArea`
	Font                       Font    `json: font`
}

type Colour struct {
	R int `json: R`
	G int `json: G`
	B int `json: B`
}

func main() {

	//The recipe file will be interpreted and assigned to the below variable based on the pdfFields struct, described above
	var pdfRecipeFromJSON PdfFields
	pdfRecipe, err := ioutil.ReadFile("pdf_recipe.json")
	if err != nil {
		fmt.Println("ERROR --> ", err)
	}
	json.Unmarshal([]byte(pdfRecipe), &pdfRecipeFromJSON)

	//Data file contains plotting and table data in an interface. To plot the data you specify the keys in the recipe, then the data file is searched for the values with that key
	var data Data
	dataToPull, err := ioutil.ReadFile("data.json")
	if err != nil {
		fmt.Println("ERROR --> ", err)
	}
	json.Unmarshal([]byte(dataToPull), &data)

	//The recipe's contents will be used to initialise the PDF, with the pdfSettings property dictating settings for the PDF, like the page orientation and margin sizes
	pdf, err := InitialisePDF(pdfRecipeFromJSON)
	if err != nil {
		fmt.Println("ERROR -->", err)
	}

	//Adding items to the pdf and processing them depending on their type (tables, vertical bar charts etc)
	ProcessPDFContentsItems(pdf, pdfRecipeFromJSON, data)

	//Recipe "pdfSettings" property is scanned for the location to save the PDF to, and for the file name that we're saving the PDF as
	err = SavePDF(pdfRecipeFromJSON, pdf)
	if err != nil {
		fmt.Println("ERROR -->", err)
	} else {
		fmt.Println()
	}

}

//////////////////////////////////////////////////////////////////////
// Parsing the recipes based on the itemType
func ProcessPDFContentsItems(pdf *gofpdf.Fpdf, contentsToProcessFromRecipe PdfFields, dataset Data) {

	for _, itemToProcess := range contentsToProcessFromRecipe.PdfContents {

		//For each itemType in the pdfContents array in the recipe file, process each recipe depending on the itemType
		switch itemToProcess.ItemType {

		case "textBlock":

			fmt.Println("Found textblock | Text --> ", itemToProcess.Text)
			err := ProcessTextBlockPDFItem(pdf, itemToProcess, contentsToProcessFromRecipe)
			if err != nil {
				fmt.Println(err)
			}

		case "table":

			fmt.Println("Found table || Data Source --> ", itemToProcess.DataSource, "-*- Data series --> ", itemToProcess.DataSeries)
			err := ProcessTablePDFItem(pdf, itemToProcess, contentsToProcessFromRecipe, dataset)
			if err != nil {
				fmt.Println(err)
			}

		case "verticalBar":

			fmt.Println("Found vertical bar chart || Data Source --> ", itemToProcess.DataSource, "-*- Data series --> ", itemToProcess.DataSeries)
			err := ProcessVerticalBarChartPDFItem(pdf, itemToProcess, contentsToProcessFromRecipe, dataset)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

//////////////////////////////////////////////////////////////////////
//Processing table
func ProcessTablePDFItem(pdf *gofpdf.Fpdf, tableItem PdfContentItem, pdfFields PdfFields, data Data) (err error) {

	font := FetchTextFormattingFromRecipe(tableItem.Font)
	tableWidth := 100.0
	if tableItem.Width >= 0.0 {
		tableWidth = tableItem.Width
	}
	columnWidth := tableWidth / 2.0

	//Settings the x and y position for the text, and making position 0 equivalent to the margin that we've set
	getXPosition := tableItem.XPosition + pdfFields.PdfSettings.PageLeftAndRightMargins
	getYPosition := tableItem.YPosition + pdfFields.PdfSettings.PageTopMargin

	//Looping through the datasets to check if the table function property 'dataseries' matches the name of a dataset
	for _, dataset := range data {
		if tableItem.DataSource == dataset.DataSource {

			//Header formatting
			pdf.SetFont(font.HeaderFont.Family, font.HeaderFont.Style, font.HeaderFont.Size)
			pdf.SetTextColor(font.HeaderFont.Colour.R, font.HeaderFont.Colour.G, font.HeaderFont.Colour.B)
			pdf.SetFillColor(font.HeaderFont.CellFill.Colour.R, font.HeaderFont.CellFill.Colour.G, font.HeaderFont.CellFill.Colour.B)
			pdf.SetDrawColor(font.HeaderFont.CellBorders.Colour.R, font.HeaderFont.CellBorders.Colour.G, font.HeaderFont.CellBorders.Colour.B)

			//Header row - Category cell
			pdf.SetXY(getXPosition, getYPosition)
			pdf.MultiCell(columnWidth, font.HeaderFont.Size+font.HeaderFont.LineSpacing, tableItem.DataSeriesCategory, font.HeaderFont.CellBorders.Style, font.HeaderFont.Alignment, font.HeaderFont.CellFill.Filled)
			//Header row - volume cell - Here we are starting from the end of the category cell
			pdf.SetXY(getXPosition+columnWidth, getYPosition)
			pdf.MultiCell(columnWidth, font.HeaderFont.Size+font.HeaderFont.LineSpacing, tableItem.DataSeries, font.HeaderFont.CellBorders.Style, font.HeaderFont.Alignment, font.HeaderFont.CellFill.Filled)

			getYPosition = getYPosition + font.HeaderFont.Size + font.HeaderFont.LineSpacing

			//Row formatting
			pdf.SetFont(font.Family, font.Style, font.Size)
			pdf.SetTextColor(font.Colour.R, font.Colour.G, font.Colour.B)
			pdf.SetFillColor(font.CellFill.Colour.R, font.CellFill.Colour.G, font.CellFill.Colour.B)
			pdf.SetDrawColor(font.CellBorders.Colour.R, font.CellBorders.Colour.G, font.CellBorders.Colour.B)

			//If the table is getting larger than the height that we've set in the recipe, break the loop and insert a row with an elipsis
			cumulativeTableHeight := font.HeaderFont.Size + font.HeaderFont.LineSpacing

			for _, point := range dataset.DataPoints {

				//For each point from the dataset, draw the category and volume cells (rows)
				pdf.SetXY(getXPosition, getYPosition)

				//If the table starts getting larger than the height set for it, stop adding rows
				if cumulativeTableHeight+0.5*(font.Size+font.LineSpacing) > tableItem.Height {
					pdf.MultiCell(tableWidth, 0.5*(font.Size+font.LineSpacing), "...", font.CellBorders.Style, "CB", font.CellFill.Filled)
					break
				}

				//Point is an interface that can hold anything - any datatype. We must assert the type first, as an MSI, then we can index it, finally convert it to a string
				pdf.MultiCell(columnWidth, font.Size+font.LineSpacing, point.(map[string]interface{})[tableItem.DataSeriesCategory].(string), font.CellBorders.Style, font.Alignment, font.CellFill.Filled)
				pdf.SetXY(getXPosition+columnWidth, getYPosition)
				pdf.MultiCell(columnWidth, font.Size+font.LineSpacing, fmt.Sprintf("%.f", point.(map[string]interface{})[tableItem.DataSeries]), font.CellBorders.Style, font.Alignment, font.CellFill.Filled)

				getYPosition = getYPosition + font.Size + font.LineSpacing
				cumulativeTableHeight = cumulativeTableHeight + font.Size + font.LineSpacing
			}
		}
	}

	return err
}

//////////////////////////////////////////////////////////////////////
//Processing vertical bar charts
func ProcessVerticalBarChartPDFItem(pdf *gofpdf.Fpdf, vbarItem PdfContentItem, pdfSettings PdfFields, data Data) (err error) {

	yAxisTicks := 5.0
	if vbarItem.ChartSettings.NumberOfYAxisTicks > 0 {
		yAxisTicks = vbarItem.ChartSettings.NumberOfYAxisTicks
	}
	//We want to count the x position and max y position as ticks marks so we take one less
	yAxisTicks = yAxisTicks - 1

	for _, dataset := range data {
		if vbarItem.DataSource == dataset.DataSource {

			font := FetchTextFormattingFromRecipe(vbarItem.ChartSettings.ChartTextFont)

			chartBoxX := vbarItem.XPosition + pdfSettings.PdfSettings.PageLeftAndRightMargins
			chartBoxY := vbarItem.YPosition + pdfSettings.PdfSettings.PageTopMargin
			chartWidth := vbarItem.Width - vbarItem.ChartSettings.DistanceFromSidesOfChartArea
			yAxisXPosition := chartBoxX + vbarItem.ChartSettings.DistanceFromSidesOfChartArea
			yAxisMaxYPosition := chartBoxY + vbarItem.ChartSettings.DistanceFromTopOfChartArea
			axisZeroPosition := chartBoxY + vbarItem.Height - vbarItem.ChartSettings.DistanceFromBottomOfChartArea
			chartHeight := axisZeroPosition - yAxisMaxYPosition

			//First we draw the charts background box container, from the chartsettings in the markup
			pdf.SetFillColor(vbarItem.ChartSettings.WatermarkFormat.FillColour.R, vbarItem.ChartSettings.WatermarkFormat.FillColour.G, vbarItem.ChartSettings.WatermarkFormat.FillColour.B)
			pdf.SetDrawColor(vbarItem.ChartSettings.WatermarkFormat.BorderColour.R, vbarItem.ChartSettings.WatermarkFormat.BorderColour.G, vbarItem.ChartSettings.WatermarkFormat.BorderColour.B)

			pdf.Rect(chartBoxX, chartBoxY, vbarItem.Width, vbarItem.Height, vbarItem.ChartSettings.WatermarkFormat.Style)

			//Getting the highest value in the dataset in order to scale the bars and set the max value on the y-axis
			maxValueFromData := 0.0
			numberCategories := 0.0
			for _, valuesFromDataPoints := range dataset.DataPoints {
				if valuesFromDataPoints.(map[string]interface{})[vbarItem.DataSeries].(float64) > maxValueFromData {
					maxValueFromData = valuesFromDataPoints.(map[string]interface{})[vbarItem.DataSeries].(float64)
				}
				numberCategories = numberCategories + 1

			}

			maxValueForYAxis := GetMaxValueForAxisOnChart(maxValueFromData)

			yAxisTickLabel := maxValueForYAxis

			//Drawing the y axis
			pdf.SetLineWidth(vbarItem.ChartSettings.AxisFormat.LineWidth)
			pdf.SetDrawColor(vbarItem.ChartSettings.AxisFormat.LineColour.R, vbarItem.ChartSettings.AxisFormat.LineColour.G, vbarItem.ChartSettings.AxisFormat.LineColour.B)
			pdf.Line(yAxisXPosition, yAxisMaxYPosition, yAxisXPosition, axisZeroPosition)
			tickIntervalOnAxis := chartHeight / yAxisTicks
			tickYPosition := yAxisMaxYPosition

			pdf.SetFont(font.Family, font.Style, font.Size)
			pdf.SetTextColor(font.Colour.R, font.Colour.G, font.Colour.B)
			tickLength := vbarItem.ChartSettings.TickMarkLength
			for i := 0.0; i <= yAxisTicks; {

				//Drawing the tick line
				pdf.SetLineWidth(vbarItem.ChartSettings.AxisFormat.LineWidth)
				pdf.SetDrawColor(vbarItem.ChartSettings.AxisFormat.LineColour.R, vbarItem.ChartSettings.AxisFormat.LineColour.G, vbarItem.ChartSettings.AxisFormat.LineColour.B)
				pdf.Line(yAxisXPosition, tickYPosition, yAxisXPosition-tickLength, tickYPosition)
				//Calculating the position of the tick labels
				////We work out the gap between the yaxis and chart box, set label to yaxis, but justify the position of the label text right
				pdf.SetXY(chartBoxX, tickYPosition-(0.5*font.Size))
				pdf.CellFormat(vbarItem.ChartSettings.DistanceFromSidesOfChartArea-vbarItem.ChartSettings.TickMarkLength, font.Size, fmt.Sprintf("%.f", yAxisTickLabel), "", 0, "RM", false, 0, "")

				tickYPosition = tickYPosition + tickIntervalOnAxis
				yAxisTickLabel = yAxisTickLabel - (maxValueForYAxis / yAxisTicks)
				i++
			}

			//Drawing x axis tick marks
			tickXInterval := (chartWidth - vbarItem.ChartSettings.DistanceFromSidesOfChartArea) / (numberCategories)
			tickXPosition := yAxisXPosition

			//Bars and x axis labels
			for _, values := range dataset.DataPoints {

				//Drawing the tick line
				pdf.SetLineWidth(vbarItem.ChartSettings.AxisFormat.LineWidth)
				pdf.SetDrawColor(vbarItem.ChartSettings.AxisFormat.LineColour.R, vbarItem.ChartSettings.AxisFormat.LineColour.G, vbarItem.ChartSettings.AxisFormat.LineColour.B)
				pdf.Line(tickXPosition, axisZeroPosition, tickXPosition, axisZeroPosition+tickLength)
				//Calculating the position of the tick labels
				pdf.SetXY(tickXPosition, axisZeroPosition+(0.5*tickLength))
				pdf.CellFormat(tickXInterval, font.Size, values.(map[string]interface{})[vbarItem.DataSeriesCategory].(string), "", 0, "CM", false, 0, "")

				//Drawing the bars
				////Bar height relative to the max value on the y axis
				barHeight := (values.(map[string]interface{})[vbarItem.DataSeries].(float64) / maxValueForYAxis) * (axisZeroPosition - yAxisMaxYPosition)
				////Bar formatting
				pdf.SetFillColor(vbarItem.ChartSettings.SeriesFormat.FillColour.R, vbarItem.ChartSettings.SeriesFormat.FillColour.G, vbarItem.ChartSettings.SeriesFormat.FillColour.B)
				pdf.SetDrawColor(vbarItem.ChartSettings.SeriesFormat.BorderColour.R, vbarItem.ChartSettings.SeriesFormat.BorderColour.G, vbarItem.ChartSettings.SeriesFormat.BorderColour.B)
				////Drawing the bars
				pdf.Rect(tickXPosition+vbarItem.ChartSettings.GapBetweenBars, axisZeroPosition, tickXInterval-(2*vbarItem.ChartSettings.GapBetweenBars), -barHeight, vbarItem.ChartSettings.SeriesFormat.Style)

				tickXPosition = tickXPosition + tickXInterval
			}
			//Drawing final tickmark on x axis
			pdf.SetLineWidth(vbarItem.ChartSettings.AxisFormat.LineWidth)
			pdf.SetDrawColor(vbarItem.ChartSettings.AxisFormat.LineColour.R, vbarItem.ChartSettings.AxisFormat.LineColour.G, vbarItem.ChartSettings.AxisFormat.LineColour.B)
			pdf.Line(tickXPosition, axisZeroPosition, tickXPosition, axisZeroPosition+tickLength)

			//Drawing the x axis line
			pdf.SetLineWidth(vbarItem.ChartSettings.AxisFormat.LineWidth)
			pdf.SetDrawColor(vbarItem.ChartSettings.AxisFormat.LineColour.R, vbarItem.ChartSettings.AxisFormat.LineColour.G, vbarItem.ChartSettings.AxisFormat.LineColour.B)
			pdf.Line(yAxisXPosition, axisZeroPosition, chartBoxX+chartWidth, axisZeroPosition)

			//Adding the chart title
			////Font formatting
			pdf.SetFont(vbarItem.ChartSettings.ChartTitle.Font.Family, vbarItem.ChartSettings.ChartTitle.Font.Style, vbarItem.ChartSettings.ChartTitle.Font.Size)
			pdf.SetFillColor(vbarItem.ChartSettings.ChartTitle.Font.CellFill.Colour.R, vbarItem.ChartSettings.ChartTitle.Font.CellFill.Colour.G, vbarItem.ChartSettings.ChartTitle.Font.CellFill.Colour.B)
			pdf.SetTextColor(vbarItem.ChartSettings.ChartTitle.Font.Colour.R, vbarItem.ChartSettings.ChartTitle.Font.Colour.G, vbarItem.ChartSettings.ChartTitle.Font.Colour.B)
			pdf.SetDrawColor(vbarItem.ChartSettings.AxisFormat.LineColour.R, vbarItem.ChartSettings.AxisFormat.LineColour.G, vbarItem.ChartSettings.AxisFormat.LineColour.B)
			////Getting the title width in order to set it's centre alignment and the width of the cell
			titleWidth := pdf.GetStringWidth(vbarItem.ChartSettings.ChartTitle.Text)
			////Setting the position of the title
			pdf.SetXY(((chartBoxX + (vbarItem.Width / 2)) - (0.5 * titleWidth)), chartBoxY+vbarItem.ChartSettings.ChartTitle.DistanceFromTopOfChartArea)
			////Adding the text
			pdf.CellFormat(titleWidth+5, vbarItem.ChartSettings.ChartTitle.Font.Size+vbarItem.ChartSettings.ChartTitle.Font.LineSpacing, vbarItem.ChartSettings.ChartTitle.Text, vbarItem.ChartSettings.ChartTitle.Font.CellBorders.Style, 0, vbarItem.ChartSettings.ChartTitle.Font.Alignment, vbarItem.ChartSettings.ChartTitle.Font.CellFill.Filled, 0, "")

		}
	}
	return err
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Working out what the max value should be on the y axis based on rounding the max value from the dataset
func GetMaxValueForAxisOnChart(maxValueFromData float64) (roundedValue float64) {
	place := 1.0
	for maxValueFromData >= place*10.0 {
		place *= 10.0
	}
	return math.Ceil(maxValueFromData/place) * place
}

//////////////////////////////////////////////////////////////////////
//Processing text blocks
func ProcessTextBlockPDFItem(pdf *gofpdf.Fpdf, textBlockItem PdfContentItem, pdfSettings PdfFields) (err error) {

	font := FetchTextFormattingFromRecipe(textBlockItem.Font)

	pdf.SetFont(font.Family, font.Style, font.Size)
	pdf.SetTextColor(font.Colour.R, font.Colour.G, font.Colour.B)
	pdf.SetFillColor(font.CellFill.Colour.R, font.CellFill.Colour.G, font.CellFill.Colour.B)
	pdf.SetDrawColor(font.CellBorders.Colour.R, font.CellBorders.Colour.G, font.CellBorders.Colour.B)

	//Settings the x and y position for the text, and making position 0 equivalent to the margin that we've set
	pdf.SetXY(textBlockItem.XPosition+pdfSettings.PdfSettings.PageLeftAndRightMargins, textBlockItem.YPosition+pdfSettings.PdfSettings.PageTopMargin)

	pdf.MultiCell(textBlockItem.Width, font.Size+font.LineSpacing, textBlockItem.Text, font.CellBorders.Style, font.Alignment, font.CellFill.Filled)

	return err
}

//////////////////////////////////////////////////////////////////////////////////////////////////
//Setting defaults font settings and where there is a font element set in the recipe, use that
func FetchTextFormattingFromRecipe(formattingFromRecipeForItem Font) (font Font) {
	//Setting some default font values
	font.Size = 9.0
	font.Style = ""
	font.Family = "Helvetica"
	font.Alignment = "CM"
	font.LineSpacing = 3.0
	font.Colour.R = 0
	font.Colour.G = 0
	font.Colour.B = 0
	font.CellFill.Filled = false
	font.CellFill.Colour.R = 255
	font.CellFill.Colour.G = 255
	font.CellFill.Colour.B = 255
	font.CellBorders.Style = "1"
	font.CellBorders.Colour.R = 0
	font.CellBorders.Colour.G = 0
	font.CellBorders.Colour.B = 0
	font.HeaderFont.Size = 10.0
	font.HeaderFont.Style = "B"
	font.HeaderFont.Family = "Helvetica"
	font.HeaderFont.Alignment = "CM"
	font.HeaderFont.LineSpacing = 3.0
	font.HeaderFont.Colour.R = 0
	font.HeaderFont.Colour.G = 0
	font.HeaderFont.Colour.B = 0
	font.HeaderFont.CellFill.Filled = false
	font.HeaderFont.CellFill.Colour.R = 213
	font.HeaderFont.CellFill.Colour.G = 213
	font.HeaderFont.CellFill.Colour.B = 213
	font.HeaderFont.CellBorders.Style = "0"
	font.HeaderFont.CellBorders.Colour.R = 0
	font.HeaderFont.CellBorders.Colour.G = 0
	font.HeaderFont.CellBorders.Colour.B = 0

	if formattingFromRecipeForItem.Size > 0.0 {
		font.Size = formattingFromRecipeForItem.Size
	}
	if formattingFromRecipeForItem.Style == "B" || formattingFromRecipeForItem.Style == "" || formattingFromRecipeForItem.Style == "I" || formattingFromRecipeForItem.Style == "BI" || formattingFromRecipeForItem.Style == "IB" {
		font.Style = formattingFromRecipeForItem.Style
	}

	if len(formattingFromRecipeForItem.Family) > 0 {
		font.Family = formattingFromRecipeForItem.Family
	}

	if formattingFromRecipeForItem.Colour.R >= 0 {
		font.Colour.R = formattingFromRecipeForItem.Colour.R
	}
	if formattingFromRecipeForItem.Colour.G >= 0 {
		font.Colour.G = formattingFromRecipeForItem.Colour.G
	}
	if formattingFromRecipeForItem.Colour.B >= 0 {
		font.Colour.B = formattingFromRecipeForItem.Colour.B
	}

	if len(formattingFromRecipeForItem.CellBorders.Style) > 0 {
		font.CellBorders.Style = formattingFromRecipeForItem.CellBorders.Style
	}

	if formattingFromRecipeForItem.CellBorders.Colour.R >= 0 {
		font.CellBorders.Colour.R = formattingFromRecipeForItem.CellBorders.Colour.R
	}
	if formattingFromRecipeForItem.CellBorders.Colour.G >= 0 {
		font.CellBorders.Colour.G = formattingFromRecipeForItem.CellBorders.Colour.G
	}
	if formattingFromRecipeForItem.CellBorders.Colour.B >= 0 {
		font.CellBorders.Colour.B = formattingFromRecipeForItem.CellBorders.Colour.B
	}

	if formattingFromRecipeForItem.CellFill.Filled == true {
		font.CellFill.Filled = formattingFromRecipeForItem.CellFill.Filled
	}

	if formattingFromRecipeForItem.CellFill.Colour.R >= 0 {
		font.CellFill.Colour.R = formattingFromRecipeForItem.CellFill.Colour.R
	}
	if formattingFromRecipeForItem.CellFill.Colour.G >= 0 {
		font.CellFill.Colour.G = formattingFromRecipeForItem.CellFill.Colour.G
	}
	if formattingFromRecipeForItem.CellFill.Colour.B >= 0 {
		font.CellFill.Colour.B = formattingFromRecipeForItem.CellFill.Colour.B
	}

	if formattingFromRecipeForItem.LineSpacing >= 0.0 {
		font.LineSpacing = formattingFromRecipeForItem.LineSpacing
	}

	if formattingFromRecipeForItem.HeaderFont.Size > 0.0 {
		font.HeaderFont.Size = formattingFromRecipeForItem.HeaderFont.Size
	}
	if formattingFromRecipeForItem.HeaderFont.Style == "B" || formattingFromRecipeForItem.HeaderFont.Style == "" || formattingFromRecipeForItem.HeaderFont.Style == "I" || formattingFromRecipeForItem.HeaderFont.Style == "BI" || formattingFromRecipeForItem.HeaderFont.Style == "IB" {
		font.HeaderFont.Style = formattingFromRecipeForItem.HeaderFont.Style
	}

	if len(formattingFromRecipeForItem.HeaderFont.Family) > 0 {
		font.HeaderFont.Family = formattingFromRecipeForItem.HeaderFont.Family
	}

	if formattingFromRecipeForItem.HeaderFont.Colour.R >= 0 {
		font.HeaderFont.Colour.R = formattingFromRecipeForItem.HeaderFont.Colour.R
	}
	if formattingFromRecipeForItem.HeaderFont.Colour.G >= 0 {
		font.HeaderFont.Colour.G = formattingFromRecipeForItem.HeaderFont.Colour.G
	}
	if formattingFromRecipeForItem.HeaderFont.Colour.B >= 0 {
		font.HeaderFont.Colour.B = formattingFromRecipeForItem.HeaderFont.Colour.B
	}

	if len(formattingFromRecipeForItem.HeaderFont.CellBorders.Style) > 0 {
		font.HeaderFont.CellBorders.Style = formattingFromRecipeForItem.HeaderFont.CellBorders.Style
	}

	if formattingFromRecipeForItem.HeaderFont.CellBorders.Colour.R >= 0 {
		font.HeaderFont.CellBorders.Colour.R = formattingFromRecipeForItem.HeaderFont.CellBorders.Colour.R
	}
	if formattingFromRecipeForItem.HeaderFont.CellBorders.Colour.G >= 0 {
		font.HeaderFont.CellBorders.Colour.G = formattingFromRecipeForItem.HeaderFont.CellBorders.Colour.G
	}
	if formattingFromRecipeForItem.HeaderFont.CellBorders.Colour.B >= 0 {
		font.HeaderFont.CellBorders.Colour.B = formattingFromRecipeForItem.HeaderFont.CellBorders.Colour.B
	}

	if formattingFromRecipeForItem.HeaderFont.CellFill.Filled == true {
		font.HeaderFont.CellFill.Filled = formattingFromRecipeForItem.HeaderFont.CellFill.Filled
	}

	if formattingFromRecipeForItem.HeaderFont.CellFill.Colour.R >= 0 {
		font.HeaderFont.CellFill.Colour.R = formattingFromRecipeForItem.HeaderFont.CellFill.Colour.R
	}
	if formattingFromRecipeForItem.HeaderFont.CellFill.Colour.G >= 0 {
		font.HeaderFont.CellFill.Colour.G = formattingFromRecipeForItem.HeaderFont.CellFill.Colour.G
	}
	if formattingFromRecipeForItem.HeaderFont.CellFill.Colour.B >= 0 {
		font.HeaderFont.CellFill.Colour.B = formattingFromRecipeForItem.HeaderFont.CellFill.Colour.B
	}

	if formattingFromRecipeForItem.LineSpacing >= 0.0 {
		font.LineSpacing = formattingFromRecipeForItem.LineSpacing
	}

	return font

}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Initialising the standard A4 pdf page, and where there are settings for the page set in the recipe, use those
func InitialisePDF(recipeFile PdfFields) (pdf *gofpdf.Fpdf, err error) {

	//Default values incase the recipe file doesn't contain the settings
	pageOrientation := "P"
	pageUnits := "pt"

	pageWidth := 595.28
	pageHeight := 841.89

	//Standard margin sizes
	leftAndRightMargin := 28.3
	topMarginPage := 42.5

	//Reading the recipe for the pdf's settings and applying if present and valid
	if recipeFile.PdfSettings.PageOrientation == "L" || recipeFile.PdfSettings.PageOrientation == "P" {
		pageOrientation = recipeFile.PdfSettings.PageOrientation
	}
	if recipeFile.PdfSettings.PageUnits == "pt" || recipeFile.PdfSettings.PageUnits == "mm" || recipeFile.PdfSettings.PageUnits == "cm" || recipeFile.PdfSettings.PageUnits == "in" {
		pageUnits = recipeFile.PdfSettings.PageUnits
	}
	if recipeFile.PdfSettings.PageWidth > 0.0 {
		pageWidth = recipeFile.PdfSettings.PageWidth
	}
	if recipeFile.PdfSettings.PageHeight > 0.0 {
		pageHeight = recipeFile.PdfSettings.PageHeight
	}
	if recipeFile.PdfSettings.PageLeftAndRightMargins >= 0.0 {
		leftAndRightMargin = recipeFile.PdfSettings.PageLeftAndRightMargins
	}
	if recipeFile.PdfSettings.PageTopMargin >= 0.0 {
		topMarginPage = recipeFile.PdfSettings.PageTopMargin
	}

	//If the orientation is landscape, switch the dimensions
	if pageOrientation == "L" {
		pageWidth = pageHeight
		pageHeight = pageWidth
	}

	//If a watermark is specified then draw a rectangle that's the size of the page
	watermarkR := -1
	watermarkG := -1
	watermarkB := -1

	if recipeFile.PdfSettings.Watermark.R >= 0 {
		watermarkR = recipeFile.PdfSettings.Watermark.R
	}
	if recipeFile.PdfSettings.Watermark.G >= 0 {
		watermarkG = recipeFile.PdfSettings.Watermark.G
	}
	if recipeFile.PdfSettings.Watermark.B >= 0 {
		watermarkB = recipeFile.PdfSettings.Watermark.B
	}

	pdf = gofpdf.New(pageOrientation, pageUnits, "A4", "")
	pdf.AddPage()
	pdf.SetMargins(leftAndRightMargin, topMarginPage, leftAndRightMargin)
	pdf.SetAutoPageBreak(true, 2.0)

	if watermarkR >= 0 {
		pdf.SetFillColor(watermarkR, watermarkG, watermarkB)
		pdf.Rect(0, 0, pageWidth, pageHeight, "F")
	}

	return pdf, err
}

//////////////////////////////////////////////////////////////////////////////////////////////////
//Saving the pdf with a default name and location, which is overwritten if present in the recipe
func SavePDF(recipeFile PdfFields, pdf *gofpdf.Fpdf) error {
	//Default location for the pdf to be saved and the name
	pdfLocation := "/Users/oliverwoodcock/Documents/Testing and Sharing/go/go/src/Tutorials/PDF/"
	pdfFileName := "example_pdf"

	if recipeFile.PdfSettings.PdfLocation != "" {
		pdfLocation = recipeFile.PdfSettings.PdfLocation
	}
	if recipeFile.PdfSettings.PdfName != "" {
		pdfFileName = recipeFile.PdfSettings.PdfName
	}

	pdfLocationToSaveAndName := pdfLocation + pdfFileName + ".pdf"

	return pdf.OutputFileAndClose(pdfLocationToSaveAndName)

}
