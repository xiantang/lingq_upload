# lingq_upload

## Purpose
This python script is to work with [LingQ](https://www.lingq.com) [API](https://www.lingq.com/apidocs/), specifically to upload a long list of audio files.

To help me better prepare for my [DELF B2](https://www.ciep.fr/en/delf-tout-public/detailed-information-the-examinations) exam I decided to listen and read ["Steve Jobs" by Walter Isaacson ](https://en.wikipedia.org/wiki/Steve_Jobs_(book)).

I legally purchased both the french Kindle book and audio book from Audible, then used [OpenAudible](https://github.com/openaudible/openaudible) and [mp3splitter-js](https://github.com/gbouthenot/mp3splitter-js) to create the audio files.

With the Kindle reader I am able to shrink the font size to a tiny size and copy and paste (_painfully_) one chapter at a time. Thankfully the [New LinqQ Reader](https://www.lingq.com/en/learn/fr/web/community/forum/updates-tips-and-known-issues/announcing-the-new-lingq-reader) makes dumping the text easier and much readable than using the legacy version. 

If there's an easier way to scrap the content "legally" from a Kindle book with latest version of Amazon Kindle Reader, I did not find one in my quick research. 


__EDIT: Publisher has limits places on Kindle Reader:__ https://twitter.com/paulwillgamble/status/1181667329798791168?s=20



Uploading an audio file was very problematic and not at all intuitive in part because of the limitations of [Django REST Framework (DRF)](https://stackoverflow.com/a/28036805/664933
).

Here's a great [article](https://goodcode.io/articles/django-rest-framework-file-upload/) for reference. 



## To Use Script
Create ``.env``, make sure it has the following parameters:

```
#APIKey="Token api_number"
APIKey="Token [https://www.lingq.com/accounts/apikey]"

# There is an address as per API POST doc that includes in substitute of 'xx' a two letter language code (ie 'fr' = french)
postAddress="https://www.lingq.com/api/v2/xx/lessons/"

# Specify private if material is copyrighted or personal
status="private"
```

Command:

```python upload.py -h```

## Thanks
[mescyn](https://www.lingq.com/en/learn/fr/web/community/forum/lingq-developer-forum/python-uploading-audio-via-api) - on Lingq Developer Forum

[beeman](https://www.lingq.com/en/learn/fr/web/community/forum/lingq-developer-forum/python-example-for-creating-a-lesson-for-a-course) - on Lingq Developer Forum

[@gbouthenot](https://github.com/gbouthenot) - for making [mp3splitter-js](https://github.com/gbouthenot/mp3splitter-js)





# command

```bash
python3 upload.py -a ~/Downloads/Princess_Diana-Cherry_Gilchrist_splitted/ -b ~/Downloads/Princess_Diana-Cherry_Gilchrist.epub -t "Princess Diana"
python3 upload.py -a ~/Downloads/Marley_and_Me-John_Grogan_splitted/ -b ~/Downloads/Marley_and_Me-John_Grogan.epub       --t "Marley and Me"
python3 upload.py -a ~/Downloads/The_Count_of_Monte_Cristo-Alexandre_Dumas_Audio/ -b ~/Downloads/The_Count_of_Monte_Cristo-Alexandre_Dumas.epub       -t "The Count of Monte Cristo"
python3 upload.py -a ~/Downloads/The_War_Of_The_Worlds-H_G_Wells_Audio/ -b ~/Downloads/The_War_Of_The_Worlds-H_G_Wells.epub       -t "The War Of The Worlds"
python3 upload.py -a  ~/Downloads/Dangerous_Game-Harris_William_splitted/ -b /home/neo/Downloads/Dangerous_Game-Harris_William.epub       -t "Dangerous Game"
python3 upload.py -a  ~/Downloads/The_Ring-Bernard_Smith_splitted/ -b /home/neo/Downloads/The_Ring-Bernard_Smith.mp3       -t "The Ring"
python3 upload.py -a  ~/Downloads/Breakfast/ -b /home/neo/Downloads/Breakfast_at_Tiffanys-Truman_Capote.epub       -t "Breakfast at Tiffany's "
python3 upload.py -a  ~/Downloads/Strangers_on_a_Train-Patricia_Highsmith_splitted/ -b ~/Downloads/Strangers_on_a_Train-Patricia_Highsmith.epub -t "Strangers on a Train"
python3 upload.py -a  ~/Downloads/What_Happened_at_Seacliffe-Denise_Kirby_splitted/ -b ~/Downloads/What_Happened_at_Seacliffe-Denise_Kirby.epub  -t "What Happened at Seacliffe"
python3 upload.py -a  ~/Downloads/The_Murder_at_the_Vicarage-Agatha_Christie_splitted/ -b ~/Downloads/The_Murder_at_the_Vicarage-Agatha_Christie.epub  -t "The Murder at the Vicarage"
```

