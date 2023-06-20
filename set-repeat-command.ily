\version "2.24.1"

setRepeatCommand =
#(define-music-function (parser location arg)(scheme?)
#{
  \applyContext #(lambda (ctx)
   (let* ((where (ly:context-property-where-defined ctx 'repeatCommands))
          (repeat-commands (ly:context-property where 'repeatCommands))
          (to-append
            (cond ((symbol? arg)
                   (list arg))
                  ((or (boolean? arg) (markup? arg))
                   (list (list 'volta arg)))
                  (else '())))
          (appended-settings (append repeat-commands to-append)))

    (ly:context-set-property! where 'repeatCommands appended-settings)))
#})