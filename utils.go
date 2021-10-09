package main

// From https://stackoverflow.com/a/9508766
var sentenceBreaks = []rune("!.?。։؞؟۔܀܁܂߹࠷࠹࠽࠾।॥၊။።፧፨᙮᜵᜶᠃᠉᥄᥅᪨᪩᪪᪫᭚᭛᭞᭟᰻᰼᱾᱿‼‽⁇⁈⁉⸮⸼꓿꘎꘏꛳꛷꡶꡷꣎꣏꤯꧈꧉꩝꩞꩟꫰꫱꯫﹒﹖﹗！．？𐩖𐩗𐽕𐽖𐽗𐽘𐽙𑁇𑁈𑂾𑂿𑃀𑃁𑅁𑅂𑅃𑇅𑇆𑇍𑇞𑇟𑈸𑈹𑈻𑈼𑊩𑑋𑑌𑗂𑗃𑗉𑗊𑗋𑗌𑗍𑗎𑗏𑗐𑗑𑗒𑗓𑗔𑗕𑗖𑗗𑙁𑙂𑜼𑜽𑜾𑩂𑩃𑪛𑪜𑱁𑱂𑻷𑻸𖩮𖩯𖫵𖬷𖬸𖭄𖺘𛲟𝪈")

func indexAnyRune(s []rune, chars []rune) int {
	for i := 0; i < len(s); i++ {
		for _, char := range chars {
			if s[i] == char {
				return i
			}
		}
	}

	return -1
}

func lastIndexAnyRune(s []rune, chars []rune) int {
	for i := len(s) - 1; i >= 0; i-- {
		for _, char := range chars {
			if s[i] == char {
				return i
			}
		}
	}

	return -1
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
