TODO
====
- don't spin when paused
- seeking with arrows across paragraphs
- pause longer on newline
- how could we handle characters consisting of >1 utf-8 codepoint? Normalization
  doesn't guarantee every character will be 1 utf-8 codepoint. Neither golang's
  range or python3 does this by default. Example: "\u0065\u0301"
- have something that says, you read x words in y minutes
- replace the sentence navigation logic with something simpler:
  preprocess the text to create a list of break indexes. Find the
  closest when navigating.
