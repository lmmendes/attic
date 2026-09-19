-- Build an AND query whose normalized lexemes match the beginning of words.
-- Starting from a tsvector keeps punctuation and quoting out of tsquery syntax.
CREATE FUNCTION attic_prefix_tsquery(search_text text)
RETURNS tsquery
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT COALESCE(
        to_tsquery(
            'english',
            string_agg(quote_literal(lexeme) || ':*', ' & ')
        ),
        ''::tsquery
    )
    FROM unnest(
        tsvector_to_array(to_tsvector('english', COALESCE(search_text, '')))
    ) AS tokens(lexeme);
$$;
