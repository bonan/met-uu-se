package main

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

const testData = `
<h3>Observations from Uppsala 2019-07-25 10:20 SNT</h3>
<table border='0' width='380' >
	<tr> 
		<td width='20px'></td>  
		<td>Temperature</td> 
		<td align='right'>29.2</td> 
		<td align='left'>&degC</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>-max last 12h</td> 
		<td align='right'>29.7</td> 
		<td align='left'>&degC</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>-min last 12h</td> 
		<td align='right'>16.4</td> 
		<td align='left'>&degC</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Wind speed</td> 
		<td align='right'>2.0</td> 
		<td align='left'>m/s</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Wind direction</td> 
		<td align='right'>326</td> 
		<td align='left'>&deg</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Air pressure</td> 
		<td align='right'>1016.3</td> 
		<td align='left'>hPa</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Air humidity</td> 
		<td align='right'>48.6</td> 
		<td align='left'>%</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Global radiation</td> 
		<td align='right'>705</td> 
		<td align='left'>W/m<sup>2</sup></td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Precipitation last hour</td> 
		<td align='right'>0.0</td> 
		<td align='left'>mm (tipping bucket)</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Precipitation last hour</td> 
		<td align='right'>0.00</td> 
		<td align='left'>mm (disdrometer)</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Precipitation 24 hours</td> 
		<td align='right'>0.0</td> 
		<td align='left'>mm (tipping bucket)</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Precipitation 24 hours</td> 
		<td align='right'>0.00</td> 
		<td align='left'>mm (disdrometer)</td> 
	</tr>
	<tr> 
		<td width='20px'></td>  
		<td>Snow depth/grass height</td> 
		<td align='right'>0</td> 
		<td align='left'>cm</td> 
	</tr>
</table>
`

func TestParse(t *testing.T) {
	htmlNode, err := html.Parse(strings.NewReader(testData))
	if err != nil {
		t.Error(err)
		return
	}
	vals, err := parse(htmlNode)
	if err != nil {
		t.Error(err)
		return
	}

	comp := []Value{
		{"temperature", "29.2", "°C"},
		{"temperature -max last 12h", "29.7", "°C"},
		{"temperature -min last 12h", "16.4", "°C"},
		{"wind speed", "2.0", "m/s"},
		{"wind direction", "326", "°"},
		{"air pressure", "1016.3", "hPa"},
		{"air humidity", "48.6", "%"},
		{"global radiation", "705", "W/m2"},
		{"precipitation last hour", "0.0", "mm (tipping bucket)"},
		{"precipitation last hour", "0.00", "mm (disdrometer)"},
		{"precipitation 24 hours", "0.0", "mm (tipping bucket)"},
		{"precipitation 24 hours", "0.00", "mm (disdrometer)"},
		{"snow depth/grass height", "0", "cm"},
	}

	if len(vals) != len(comp) {
		t.Error("invalid number of items")
	}

	for i, v := range comp {
		v1 := v
		v2 := vals[i]

		t.Run("test "+v.Name, func(t *testing.T) {
			if v1.Name != v2.Name {
				t.Errorf("[%s] %s != %s", v1.Name, v1.Name, v2.Name)
			}
			if v1.Value != v2.Value {
				t.Errorf("[%s] %s != %s", v1.Name, v1.Value, v2.Value)
			}
			if v1.Unit != v2.Unit {
				t.Errorf("[%s] %s != %s", v1.Name, v1.Unit, v2.Unit)
			}
		})
	}

}
