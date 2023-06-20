\version "2.24.0"
\include "set-repeat-command.ily"
\header { tagline = #f }

\score {
  \header {
      piece = "Repeat"
  }
  \new Staff{
    \time 4/4 \key c \major
    \setRepeatCommand #'start-repeat c'1 | d'1 \bar "||" e'1 | f'1 \setRepeatCommand #'end-repeat g'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Double repeats"
  }
  \new Staff{
    \time 4/4 \key c \major
    \setRepeatCommand #'start-repeat c'1 | d'1 \setRepeatCommand #'end-repeat \setRepeatCommand #'start-repeat e'1 \setRepeatCommand #'end-repeat \setRepeatCommand #'start-repeat f'1 \setRepeatCommand #'end-repeat
  }
}
\score {
  \header {
      piece = "Voltas"
  }
  \new Staff{
    \time 4/4 \key c \major
    \setRepeatCommand #'start-repeat c'1 | \setRepeatCommand #"1" d'1 \setRepeatCommand #'end-repeat \setRepeatCommand #"2" e'1 \bar "||" \setRepeatCommand ##f f'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Voltas Double Bar"
  }
  \new Staff{
    \time 4/4 \key c \major
    \setRepeatCommand #'start-repeat c'1 | \setRepeatCommand #"1" d'1 \setRepeatCommand #'end-repeat \setRepeatCommand #"2" e'1 \bar "||" \setRepeatCommand ##f f'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Voltas End"
  }
  \new Staff{
    \time 4/4 \key c \major
    \setRepeatCommand #'start-repeat c'1 | \setRepeatCommand #"1" d'1 \setRepeatCommand #'end-repeat \setRepeatCommand #"2" e'1 | f'1 \setRepeatCommand ##f \bar "|."
  }
}
\score {
  \header {
      piece = "Segno Coda"
  }
  \new Staff{
    \time 4/4 \key c \major
    | c'1 | d'1 \segnoMark 1  | e'1 \codaMark 1  \bar "||" f'1 \segnoMark 1  \bar "||" c'1 \codaMark 1  | d'1 \bar "|."
  }
}
\score {
  \header {
      piece = "Multiple Repeats"
  }
  \new Staff{
    \time 4/4 \key c \major
    \setRepeatCommand #'start-repeat c'1 | d'1 \setRepeatCommand #'end-repeat \break
    \setRepeatCommand #'start-repeat c'1 | d'1 \setRepeatCommand #'end-repeat \break
    \setRepeatCommand #'start-repeat c'1 | d'1 \setRepeatCommand #'end-repeat
  }
}
\score {
  \header {
      piece = "Double Bar and Repeat"
  }
  \new Staff{
    \time 4/4 \key c \major
    | c'1 | d'1 \bar ".|:-||" \break
    \setRepeatCommand #'start-repeat c'1 | d'1 \setRepeatCommand #'end-repeat
  }
}
