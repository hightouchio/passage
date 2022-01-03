FROM ruby:3

RUN gem install httparty

ADD test-normal.rb /test-normal.rb
RUN chmod +x /test-normal.rb

ADD test-reverse.rb /test-reverse.rb
RUN chmod +x /test-reverse.rb