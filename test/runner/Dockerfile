FROM ruby:3

RUN gem install httparty

ADD run.rb /run.rb
RUN chmod +x /run.rb
CMD /run.rb