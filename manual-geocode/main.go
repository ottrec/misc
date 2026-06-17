package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/ottrec/website/pkg/ottrecdl"
	"github.com/ottrec/website/pkg/ottrecidx"
)

func main() {
	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	context.AfterFunc(ctx, func() {
		fmt.Fprintf(os.Stderr, "\ninterrupted\n")
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		os.Exit(1)
	})

	ocl := &ottrecdl.Client{
		Base: "https://data.ottrec.ca/",
	}

	pb, err := ocl.Latest(ctx, "pb")
	if err != nil {
		panic(err)
	}

	idx, err := new(ottrecidx.Indexer).Load(pb)
	if err != nil {
		panic(err)
	}

	ctx, cancel := chromedp.NewExecAllocator(ctx, slices.Concat(chromedp.DefaultExecAllocatorOptions[:], []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false),
	})...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	latLngCh := make(chan [2]float64)
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate("https://maps.google.com"), // note: enable satellite, globe view
		listenCopyLatLng(func(lat, lng float64) {
			select {
			case latLngCh <- [2]float64{lat, lng}:
			default:
			}
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for fac := range idx.Data().Facilities() {
				addr := strings.ReplaceAll(strings.TrimSpace(fac.GetAddress()), "\n", ", ")
				fmt.Printf("// %s (%s)\n", fac.GetName(), addr)

				if _, _, ok := manualGeocode("", addr); ok {
					continue
				}

				if err := search(ctx, addr); err != nil {
					return err
				}

				tmp, _, _ := strings.Cut(addr, ",")
				tmp = strings.TrimSpace(tmp)
				tmp = strings.TrimRightFunc(tmp, unicode.IsLower)
				latLng := <-latLngCh
				fmt.Printf("case strings.HasPrefix(addr, %q): return %.5f, %.5f, true\n", tmp, latLng[0], latLng[1])
			}
			return nil
		}),
	}); err != nil {
		panic(err)
	}
}

func listenCopyLatLng(fn func(lat, lng float64)) chromedp.Action {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			latLngRe := regexp.MustCompile(`^(-?[0-9]+\.[0-9]+), (-?[0-9]+\.[0-9]+)$`)
			chromedp.ListenTarget(ctx, func(ev any) {
				switch ev := ev.(type) {
				case *runtime.EventBindingCalled:
					if ev.Name == "interceptClipboardWrite" {
						if m := latLngRe.FindStringSubmatch(ev.Payload); m != nil {
							lat, _ := strconv.ParseFloat(m[1], 64)
							lng, _ := strconv.ParseFloat(m[2], 64)
							if fn != nil {
								fn(lat, lng)
							}
						}
					}
				}
			})
			return nil
		}),
		runtime.AddBinding("interceptClipboardWrite"),
		chromedp.Evaluate(`navigator.clipboard.writeText = async text => globalThis.interceptClipboardWrite(text)`, nil),
	}
}

func search(ctx context.Context, q string) error {
	buf, _ := json.Marshal(q)
	return chromedp.Evaluate(`document.querySelector("[role=search] input[name=q]").value = `+string(buf)+`; document.querySelector("[role=search] button[aria-label=Search]").click()`, nil).Do(ctx)
}

// manualGeocode contains manual overrides for geocoding certain addresses. The
// coordinates are generally over the main entrance to each facility.
func manualGeocode(name, addr string) (lat, lng float64, ok bool) {
	switch {
	case strings.HasPrefix(addr, "1560 Heatherington R"):
		return 45.37313, -75.64800, true
	case strings.HasPrefix(addr, "960 Silver S"):
		return 45.38058, -75.73129, true
	case strings.HasPrefix(addr, "2300 Community W"):
		return 45.13019, -75.71086, true
	case strings.HasPrefix(addr, "318 Aquaview D"):
		return 45.45308, -75.47793, true
	case strings.HasPrefix(addr, "175 Woodridge C"):
		return 45.35113, -75.80940, true
	case strings.HasPrefix(addr, "2130 Radford C"):
		return 45.45895, -75.59895, true
	case strings.HasPrefix(addr, "8720 Russell R"):
		return 45.38362, -75.33728, true
	case strings.HasPrefix(addr, "2679 Innes R"):
		return 45.43419, -75.56348, true
	case strings.HasPrefix(addr, "1002 Beaverbrook L"):
		return 45.32894, -75.90115, true
	case strings.HasPrefix(addr, "50 Cassidy R"):
		return 45.32489, -75.81070, true
	case strings.HasPrefix(addr, "2915 Haughton A"):
		return 45.35996, -75.80413, true
	case strings.HasPrefix(addr, "101 Centrepointe D"):
		return 45.34433, -75.76213, true
	case strings.HasPrefix(addr, "309 McArthur R"):
		return 45.43255, -75.65549, true
	case strings.HasPrefix(addr, "2100 Cabot S"):
		return 45.38970, -75.67267, true
	case strings.HasPrefix(addr, "1490 Youville D"):
		return 45.46641, -75.54499, true
	case strings.HasPrefix(addr, "100 Brewer W"):
		return 45.38947, -75.69154, true
	case strings.HasPrefix(addr, "63 Bluegrass D"):
		return 45.28498, -75.86099, true
	case strings.HasPrefix(addr, "2185 Arch S"):
		switch {
		case strings.Contains(name, "Jim Tubman"):
			return 45.39004, -75.62967, true
		default:
			return 45.39077, -75.63046, true
		}
	case strings.HasPrefix(addr, "1500 Shea R"):
		return 45.26340, -75.90766, true
	case strings.HasPrefix(addr, "1665 Apeldoorn A"):
		return 45.35941, -75.70299, true
	case strings.HasPrefix(addr, "1520 Caldwell A"):
		return 45.37245, -75.73895, true
	case strings.HasPrefix(addr, "321 King Edward A"):
		return 45.43060, -75.68696, true
	case strings.HasPrefix(addr, "424 Chapman Mills D"):
		return 45.27210, -75.72315, true
	case strings.HasPrefix(addr, "30 Wessex R"):
		return 45.27344, -75.75230, true
	case strings.HasPrefix(addr, "345 Richmond R"):
		return 45.39202, -75.75434, true
	case strings.HasPrefix(addr, "262 Len Purcell D"):
		return 45.49924, -76.09345, true
	case strings.HasPrefix(addr, "61 Corkstown R"):
		return 45.34605, -75.82720, true
	case strings.HasPrefix(addr, "56 Fieldrow S"):
		return 45.34445, -75.73781, true
	case strings.HasPrefix(addr, "2940 Old Montreal R"):
		return 45.51783, -75.39114, true
	case strings.HasPrefix(addr, "1300 Kitchener A"):
		return 45.36755, -75.65724, true
	case strings.HasPrefix(addr, "1895 Russell R"):
		return 45.40259, -75.62747, true
	case strings.HasPrefix(addr, "363 Lorry Greenberg D"):
		return 45.36327, -75.63532, true
	case strings.HasPrefix(addr, "411 Dovercourt A"):
		return 45.38333, -75.75285, true
	case strings.HasPrefix(addr, "2020 Ogilvie R"):
		return 45.43780, -75.60174, true
	case strings.HasPrefix(addr, "2 Eaton S"):
		return 45.32643, -75.81681, true
	case strings.HasPrefix(addr, "65 Stonehaven D"):
		return 45.29050, -75.85773, true
	case strings.HasPrefix(addr, "3080 Richmond R"):
		return 45.34906, -75.80216, true
	case strings.HasPrefix(addr, "679 Deancourt C"):
		return 45.48141, -75.48676, true
	case strings.HasPrefix(addr, "250 Holland A"):
		return 45.39516, -75.73087, true
	case strings.HasPrefix(addr, "1065 Ramsey C"):
		return 45.34967, -75.79442, true
	case strings.HasPrefix(addr, "2263 Portobello B"):
		return 45.45421, -75.46367, true
	case strings.HasPrefix(addr, "3280 Leitrim R"):
		return 45.33142, -75.59795, true
	case strings.HasPrefix(addr, "107 Chesterton D"):
		return 45.35168, -75.72046, true
	case strings.HasPrefix(addr, "43 Ste-Cécile S"),
		strings.HasPrefix(addr, "43 Ste-Cecile S"):
		return 45.44236, -75.66932, true
	case strings.HasPrefix(addr, "175 Third A"):
		return 45.40211, -75.69145, true
	case strings.HasPrefix(addr, "186 Morrena R"):
		return 45.29608, -75.88526, true
	case strings.HasPrefix(addr, "70 Castlefrank R"):
		return 45.29534, -75.88503, true
	case strings.HasPrefix(addr, "1448 Meadow D"):
		return 45.26116, -75.55728, true
	case strings.HasPrefix(addr, "1480 Heron R"):
		return 45.37910, -75.65403, true
	case strings.HasPrefix(addr, "1064 Wellington Street W"):
		return 45.40368, -75.72448, true
	case strings.HasPrefix(addr, "1765 Merivale R"):
		return 45.34190, -75.72700, true
	case strings.HasPrefix(addr, "3320 Paul Anka D"):
		return 45.35191, -75.67276, true
	case strings.HasPrefix(addr, "941 Clyde A"):
		return 45.37477, -75.74637, true
	case strings.HasPrefix(addr, "10 McKitrick D"):
		return 45.29375, -75.88425, true
	case strings.HasPrefix(addr, "320 Jack Purcell L"):
		return 45.41583, -75.68942, true
	case strings.HasPrefix(addr, "1265 Walkley R"):
		return 45.37293, -75.65943, true
	case strings.HasPrefix(addr, "2500 Campeau D"):
		return 45.32146, -75.89541, true
	case strings.HasPrefix(addr, "10 Warner-Colpitts L"):
		return 45.26080, -75.92623, true
	case strings.HasPrefix(addr, "70 Aird P"):
		return 45.31114, -75.89922, true
	case strings.HasPrefix(addr, "1606 Old Wellington S"):
		return 45.14902, -75.64814, true
	case strings.HasPrefix(addr, "64 Chimo D"):
		return 45.31015, -75.88949, true
	case strings.HasPrefix(addr, "700 Longfields D"):
		return 45.28203, -75.74239, true
	case strings.HasPrefix(addr, "3242 York's Corners R"):
		return 45.22914, -75.41541, true
	case strings.HasPrefix(addr, "1525 Princess P"):
		return 45.40075, -75.68236, true
	case strings.HasPrefix(addr, "76 Larkin D"):
		return 45.28249, -75.76286, true
	case strings.HasPrefix(addr, "15 Rockcliffe W"):
		return 45.44447, -75.67495, true
	case strings.HasPrefix(addr, "200 Glen Park D"):
		return 45.43011, -75.56333, true
	case strings.HasPrefix(addr, "40 Cobourg S"):
		return 45.43464, -75.68127, true
	case strings.HasPrefix(addr, "100 Thornwood R"):
		return 45.45092, -75.65706, true
	case strings.HasPrefix(addr, "5572 Doctor Leach D"):
		return 45.22229, -75.68659, true
	case strings.HasPrefix(addr, "68 Knoxdale R"):
		return 45.33017, -75.76112, true
	case strings.HasPrefix(addr, "180 Percy S"):
		return 45.40931, -75.70174, true
	case strings.HasPrefix(addr, "2785 8th L"):
		return 45.22967, -75.46917, true
	case strings.HasPrefix(addr, "2955 Michele D"):
		return 45.35457, -75.80198, true
	case strings.HasPrefix(addr, "3500 Cambrian R"):
		return 45.25277, -75.73427, true
	case strings.HasPrefix(addr, "1295 Colonial R"):
		return 45.42134, -75.42137, true
	case strings.HasPrefix(addr, "35 Stafford R"):
		return 45.32875, -75.82226, true
	case strings.HasPrefix(addr, "16 Rowley A"):
		return 45.34923, -75.74144, true
	case strings.HasPrefix(addr, "1701 Woodroffe A"):
		switch {
		case strings.Contains(addr, "Entrance 3"): // entrance on the east side
			return 45.327259, -75.744704, true
		default:
			return 45.326930, -75.745970, true
		}
	case strings.HasPrefix(addr, "61 Main S"):
		return 45.41291, -75.67999, true
	case strings.HasPrefix(addr, "5660 Osgoode Main S"):
		return 45.14708, -75.60236, true
	case strings.HasPrefix(addr, "260 Sunnyside A"):
		return 45.39477, -75.68186, true
	case strings.HasPrefix(addr, "33 Quill S"):
		return 45.42551, -75.65704, true
	case strings.HasPrefix(addr, "4355 Halmont D"):
		return 45.42871, -75.61962, true
	case strings.HasPrefix(addr, "2250 Torquay A"):
		return 45.34784, -75.77391, true
	case strings.HasPrefix(addr, "270 Pinhey's Point R"):
		return 45.44035, -75.95391, true
	case strings.HasPrefix(addr, "930 Somerset Street W"):
		return 45.40784, -75.71481, true
	case strings.HasPrefix(addr, "1115 Dunning R"):
		return 45.51445, -75.40368, true
	case strings.HasPrefix(addr, "1585 Tenth Line R"):
		return 45.47193, -75.49286, true
	case strings.HasPrefix(addr, "4101 Innovation D"):
		return 45.34069, -75.93026, true
	case strings.HasPrefix(addr, "300 Des Pères-Blancs A"),
		strings.HasPrefix(addr, "300 Des Peres-Blancs A"):
		return 45.44386, -75.65978, true
	case strings.HasPrefix(addr, "6095 Perth S"):
		return 45.19540, -75.83777, true
	case strings.HasPrefix(addr, "4310 Shoreline D"):
		return 45.27732, -75.68748, true
	case strings.HasPrefix(addr, "380 Springfield R"):
		return 45.45076, -75.67858, true
	case strings.HasPrefix(addr, "102 Greenview A"):
		return 45.36376, -75.80182, true
	case strings.HasPrefix(addr, "172 Guigues A"):
		return 45.43217, -75.69127, true
	case strings.HasPrefix(addr, "60 Mann A"):
		return 45.41973, -75.67447, true
	case strings.HasPrefix(addr, "250 Somerset Street E"):
		return 45.42289, -75.67751, true
	case strings.HasPrefix(addr, "3380 D'Aoust A"):
		return 45.34969, -75.63622, true
	case strings.HasPrefix(addr, "245 Centrum B"):
		return 45.48059, -75.51152, true
	case strings.HasPrefix(addr, "998 Valin S"):
		return 45.47129, -75.46205, true
	case strings.HasPrefix(addr, "220 Stoneway D"):
		return 45.28812, -75.71598, true
	case strings.HasPrefix(addr, "2040 Ogilvie R"):
		return 45.43741, -75.60067, true
	case strings.HasPrefix(addr, "525 Côté S"),
		strings.HasPrefix(addr, "525 Cote S"):
		return 45.43640, -75.64706, true
	case strings.HasPrefix(addr, "30 Woodfield D"):
		return 45.33662, -75.73042, true
	case strings.HasPrefix(addr, "2960 Riverside D"):
		return 45.36977, -75.69137, true
	case strings.HasPrefix(addr, "141 Bayview Station R"):
		return 45.40779, -75.72285, true
	case strings.HasPrefix(addr, "100 Charlie Rogers P"):
		return 45.29458, -75.90132, true
	case strings.HasPrefix(addr, "7950 Lawrence S"):
		return 45.16328, -75.45480, true
	case strings.HasPrefix(addr, "3832 Carp R"):
		return 45.34921, -76.03922, true
	case strings.HasPrefix(addr, "100 Malvern D"):
		return 45.28024, -75.76215, true
	case strings.HasPrefix(addr, "681 Seyton D"):
		return 45.31543, -75.83544, true
	case strings.HasPrefix(addr, "50 Bellman D"):
		return 45.32697, -75.78313, true
	case strings.HasPrefix(addr, "821 March R"):
		return 45.35558, -75.93364, true
	case strings.HasPrefix(addr, "19 Leeming D"):
		return 45.34957, -75.83264, true
	case strings.HasPrefix(addr, "7 Sycamore D"):
		return 45.31956, -75.82174, true
	case strings.HasPrefix(addr, "51 Stonehurst A"):
		return 45.40846, -75.72900, true
	case strings.HasPrefix(addr, "1560 Clover S"):
		return 45.37929, -75.67710, true
	case strings.HasPrefix(addr, "135 Craig Henry D"):
		return 45.33334, -75.77464, true
	case strings.HasPrefix(addr, "3739 Carp R"):
		return 45.34454, -76.03612, true
	case strings.HasPrefix(addr, "5717 Rockdale R"):
		return 45.35600, -75.35150, true
	}
	return 0, 0, false
}
