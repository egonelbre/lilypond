\version "2.24.1"
\header { tagline = #f }

\score {
  \header {
      piece = "Repeat"
  }
  \new Staff{
  \accidentalStyle modern
    \time 4/4 \key c \major
    \set Score.repeatCommands = #'(start-repeat) c'1 | d'1 \bar "||" e'1 | f'1 \set Score.repeatCommands = #'(end-repeat) g'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Double repeats"
  }
  \new Staff{
  \accidentalStyle modern
    \time 4/4 \key c \major
    \set Score.repeatCommands = #'(start-repeat) c'1 | d'1 \set Score.repeatCommands = #'(end-repeat start-repeat) e'1 \set Score.repeatCommands = #'(end-repeat start-repeat) f'1 \set Score.repeatCommands = #'(end-repeat)
  }
}
\score {
  \header {
      piece = "Voltas"
  }
  \new Staff{
  \accidentalStyle modern
    \time 4/4 \key c \major
    \set Score.repeatCommands = #'(start-repeat) c'1 | \set Score.repeatCommands = #'((volta "1")) d'1 \set Score.repeatCommands = #'(end-repeat (volta "2")) e'1 \bar "||" \set Score.repeatCommands = #'((volta #f)) f'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Voltas Double Bar"
  }
  \new Staff{
  \accidentalStyle modern
    \time 4/4 \key c \major
    \set Score.repeatCommands = #'(start-repeat) c'1 | \set Score.repeatCommands = #'((volta "1")) d'1 \set Score.repeatCommands = #'(end-repeat (volta "2")) e'1 \bar "||" \set Score.repeatCommands = #'((volta #f)) f'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Voltas End"
  }
  \new Staff{
  \accidentalStyle modern
    \time 4/4 \key c \major
    \set Score.repeatCommands = #'(start-repeat) c'1 | \set Score.repeatCommands = #'((volta "1")) d'1 \set Score.repeatCommands = #'(end-repeat (volta "2")) e'1 | f'1 \set Score.repeatCommands = #'((volta #f)) \bar "|."
  }
}
\score {
  \header {
      piece = "Segno Coda"
  }
  \new Staff{
  \accidentalStyle modern
    \time 4/4 \key c \major
    | c'1 | d'1 \segnoMark 1  | e'1 \codaMark 1  \bar "||" f'1 \segnoMark 1  \bar "||" c'1 \codaMark 1  | d'1 \bar "|."
  }
}
